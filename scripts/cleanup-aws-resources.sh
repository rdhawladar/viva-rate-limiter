#!/bin/bash

# AWS Cleanup Script for Viva Rate Limiter
# This script removes all AWS resources associated with the project

set -e

# Configuration
AWS_REGION="ap-southeast-1"
ECR_REPOSITORY="viva-rate-limiter"
CLUSTERS=("viva-cluster-dev" "viva-cluster-stage")
NAMESPACES=("dev" "stage")

echo "======================================"
echo "AWS Resources Cleanup Script"
echo "Region: $AWS_REGION"
echo "======================================"
echo ""
echo "WARNING: This will delete ALL AWS resources for the Viva Rate Limiter project!"
echo "This includes EKS clusters, ECR repositories, VPCs, and associated resources."
echo ""
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Cleanup cancelled."
    exit 0
fi

# Function to check if AWS CLI is configured
check_aws_cli() {
    if ! command -v aws &> /dev/null; then
        echo "AWS CLI is not installed. Please install it first."
        exit 1
    fi
    
    if ! aws sts get-caller-identity &> /dev/null; then
        echo "AWS CLI is not configured. Please configure it with 'aws configure'"
        exit 1
    fi
}

# Function to delete Kubernetes resources
delete_k8s_resources() {
    local cluster_name=$1
    local namespace=$2
    
    echo "Attempting to delete Kubernetes resources in cluster: $cluster_name, namespace: $namespace"
    
    # Update kubeconfig
    if aws eks update-kubeconfig --region $AWS_REGION --name $cluster_name 2>/dev/null; then
        # Delete all resources in namespace
        kubectl delete all --all -n $namespace 2>/dev/null || true
        kubectl delete configmap --all -n $namespace 2>/dev/null || true
        kubectl delete secret --all -n $namespace 2>/dev/null || true
        kubectl delete pvc --all -n $namespace 2>/dev/null || true
        kubectl delete namespace $namespace 2>/dev/null || true
        echo "✓ Deleted Kubernetes resources in $namespace"
    else
        echo "⚠ Could not connect to cluster $cluster_name (may not exist)"
    fi
}

# Function to delete EKS cluster
delete_eks_cluster() {
    local cluster_name=$1
    
    echo "Checking for EKS cluster: $cluster_name"
    
    if aws eks describe-cluster --name $cluster_name --region $AWS_REGION &>/dev/null; then
        echo "Deleting EKS cluster: $cluster_name"
        
        # Delete node groups first
        nodegroups=$(aws eks list-nodegroups --cluster-name $cluster_name --region $AWS_REGION --query 'nodegroups[]' --output text 2>/dev/null || true)
        for nodegroup in $nodegroups; do
            echo "  Deleting node group: $nodegroup"
            aws eks delete-nodegroup --cluster-name $cluster_name --nodegroup-name $nodegroup --region $AWS_REGION 2>/dev/null || true
            
            # Wait for node group deletion
            echo "  Waiting for node group deletion..."
            aws eks wait nodegroup-deleted --cluster-name $cluster_name --nodegroup-name $nodegroup --region $AWS_REGION 2>/dev/null || true
        done
        
        # Delete the cluster
        aws eks delete-cluster --name $cluster_name --region $AWS_REGION
        echo "  Waiting for cluster deletion (this may take several minutes)..."
        aws eks wait cluster-deleted --name $cluster_name --region $AWS_REGION 2>/dev/null || true
        echo "✓ Deleted EKS cluster: $cluster_name"
    else
        echo "⚠ EKS cluster $cluster_name not found"
    fi
}

# Function to delete ECR repository
delete_ecr_repository() {
    local repo_name=$1
    
    echo "Checking for ECR repository: $repo_name"
    
    if aws ecr describe-repositories --repository-names $repo_name --region $AWS_REGION &>/dev/null; then
        echo "Deleting ECR repository: $repo_name"
        aws ecr delete-repository --repository-name $repo_name --region $AWS_REGION --force
        echo "✓ Deleted ECR repository: $repo_name"
    else
        echo "⚠ ECR repository $repo_name not found"
    fi
}

# Function to delete VPC and networking resources
delete_vpc_resources() {
    echo "Looking for VPCs with EKS tags..."
    
    # Find VPCs tagged for EKS
    vpcs=$(aws ec2 describe-vpcs --region $AWS_REGION \
        --filters "Name=tag:kubernetes.io/cluster/viva-cluster-dev,Values=owned,shared" \
                  "Name=tag:kubernetes.io/cluster/viva-cluster-stage,Values=owned,shared" \
        --query 'Vpcs[].VpcId' --output text 2>/dev/null || true)
    
    for vpc_id in $vpcs; do
        echo "Processing VPC: $vpc_id"
        
        # Delete NAT Gateways
        nat_gateways=$(aws ec2 describe-nat-gateways --region $AWS_REGION \
            --filter "Name=vpc-id,Values=$vpc_id" "Name=state,Values=available" \
            --query 'NatGateways[].NatGatewayId' --output text 2>/dev/null || true)
        
        for nat_id in $nat_gateways; do
            echo "  Deleting NAT Gateway: $nat_id"
            aws ec2 delete-nat-gateway --nat-gateway-id $nat_id --region $AWS_REGION 2>/dev/null || true
        done
        
        # Delete Internet Gateways
        igws=$(aws ec2 describe-internet-gateways --region $AWS_REGION \
            --filters "Name=attachment.vpc-id,Values=$vpc_id" \
            --query 'InternetGateways[].InternetGatewayId' --output text 2>/dev/null || true)
        
        for igw_id in $igws; do
            echo "  Detaching and deleting Internet Gateway: $igw_id"
            aws ec2 detach-internet-gateway --internet-gateway-id $igw_id --vpc-id $vpc_id --region $AWS_REGION 2>/dev/null || true
            aws ec2 delete-internet-gateway --internet-gateway-id $igw_id --region $AWS_REGION 2>/dev/null || true
        done
        
        # Delete subnets
        subnets=$(aws ec2 describe-subnets --region $AWS_REGION \
            --filters "Name=vpc-id,Values=$vpc_id" \
            --query 'Subnets[].SubnetId' --output text 2>/dev/null || true)
        
        for subnet_id in $subnets; do
            echo "  Deleting Subnet: $subnet_id"
            aws ec2 delete-subnet --subnet-id $subnet_id --region $AWS_REGION 2>/dev/null || true
        done
        
        # Delete route tables (except main)
        route_tables=$(aws ec2 describe-route-tables --region $AWS_REGION \
            --filters "Name=vpc-id,Values=$vpc_id" \
            --query 'RouteTables[?Associations[0].Main != `true`].RouteTableId' --output text 2>/dev/null || true)
        
        for rt_id in $route_tables; do
            echo "  Deleting Route Table: $rt_id"
            aws ec2 delete-route-table --route-table-id $rt_id --region $AWS_REGION 2>/dev/null || true
        done
        
        # Delete security groups (except default)
        security_groups=$(aws ec2 describe-security-groups --region $AWS_REGION \
            --filters "Name=vpc-id,Values=$vpc_id" \
            --query 'SecurityGroups[?GroupName != `default`].GroupId' --output text 2>/dev/null || true)
        
        for sg_id in $security_groups; do
            echo "  Deleting Security Group: $sg_id"
            aws ec2 delete-security-group --group-id $sg_id --region $AWS_REGION 2>/dev/null || true
        done
        
        # Finally, delete the VPC
        echo "  Deleting VPC: $vpc_id"
        aws ec2 delete-vpc --vpc-id $vpc_id --region $AWS_REGION 2>/dev/null || true
        echo "✓ Deleted VPC: $vpc_id"
    done
}

# Function to delete IAM roles and policies
delete_iam_resources() {
    echo "Looking for IAM roles with EKS prefix..."
    
    # Find IAM roles for EKS
    roles=$(aws iam list-roles --query "Roles[?contains(RoleName, 'eks') || contains(RoleName, 'viva')].RoleName" --output text 2>/dev/null || true)
    
    for role_name in $roles; do
        echo "Processing IAM role: $role_name"
        
        # Detach policies
        policies=$(aws iam list-attached-role-policies --role-name $role_name --query 'AttachedPolicies[].PolicyArn' --output text 2>/dev/null || true)
        for policy_arn in $policies; do
            echo "  Detaching policy: $policy_arn"
            aws iam detach-role-policy --role-name $role_name --policy-arn $policy_arn 2>/dev/null || true
        done
        
        # Delete inline policies
        inline_policies=$(aws iam list-role-policies --role-name $role_name --query 'PolicyNames[]' --output text 2>/dev/null || true)
        for policy_name in $inline_policies; do
            echo "  Deleting inline policy: $policy_name"
            aws iam delete-role-policy --role-name $role_name --policy-name $policy_name 2>/dev/null || true
        done
        
        # Delete the role
        echo "  Deleting role: $role_name"
        aws iam delete-role --role-name $role_name 2>/dev/null || true
        echo "✓ Deleted IAM role: $role_name"
    done
}

# Main cleanup process
echo ""
echo "Starting cleanup process..."
echo "======================================"

# Check AWS CLI
check_aws_cli

# Step 1: Delete Kubernetes resources
echo ""
echo "Step 1: Deleting Kubernetes resources..."
for i in ${!CLUSTERS[@]}; do
    delete_k8s_resources "${CLUSTERS[$i]}" "${NAMESPACES[$i]}"
done

# Step 2: Delete EKS clusters
echo ""
echo "Step 2: Deleting EKS clusters..."
for cluster in "${CLUSTERS[@]}"; do
    delete_eks_cluster "$cluster"
done

# Step 3: Delete ECR repository
echo ""
echo "Step 3: Deleting ECR repository..."
delete_ecr_repository "$ECR_REPOSITORY"

# Step 4: Delete VPC and networking resources
echo ""
echo "Step 4: Deleting VPC and networking resources..."
delete_vpc_resources

# Step 5: Delete IAM roles and policies
echo ""
echo "Step 5: Deleting IAM roles and policies..."
delete_iam_resources

echo ""
echo "======================================"
echo "Cleanup process completed!"
echo ""
echo "Note: Some resources may take a few minutes to fully delete."
echo "You can verify all resources are deleted in the AWS Console."
echo ""
echo "Resources that were targeted for deletion:"
echo "- EKS Clusters: ${CLUSTERS[*]}"
echo "- ECR Repository: $ECR_REPOSITORY"
echo "- Associated VPCs, subnets, security groups"
echo "- IAM roles and policies with 'eks' or 'viva' prefix"