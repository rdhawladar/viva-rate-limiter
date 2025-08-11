#\!/bin/bash

# Deployment script for Viva Rate Limiter
# Usage: ./deploy.sh <environment> <action>
# Example: ./deploy.sh dev apply

set -e

ENVIRONMENT=$1
ACTION=$2

if [ -z "$ENVIRONMENT" ] || [ -z "$ACTION" ]; then
    echo "Usage: ./deploy.sh <environment> <action>"
    echo "Environments: dev, stage, prod"
    echo "Actions: init, plan, apply, destroy"
    exit 1
fi

if [[ ! "$ENVIRONMENT" =~ ^(dev|stage|prod)$ ]]; then
    echo "Invalid environment. Must be: dev, stage, prod"
    exit 1
fi

if [[ ! "$ACTION" =~ ^(init|plan|apply|destroy)$ ]]; then
    echo "Invalid action. Must be: init, plan, apply, destroy"
    exit 1
fi

echo "🚀 Deploying to $ENVIRONMENT environment..."

cd terraform

case $ACTION in
    init)
        echo "Initializing Terraform for $ENVIRONMENT..."
        terraform init \
            -backend-config="key=eks/${ENVIRONMENT}/terraform.tfstate" \
            -reconfigure
        ;;
    plan)
        echo "Planning changes for $ENVIRONMENT..."
        terraform plan \
            -var-file="environments/${ENVIRONMENT}/terraform.tfvars"
        ;;
    apply)
        echo "Applying changes to $ENVIRONMENT..."
        terraform apply \
            -var-file="environments/${ENVIRONMENT}/terraform.tfvars" \
            -auto-approve
        
        echo "Updating kubeconfig..."
        aws eks update-kubeconfig \
            --region ap-southeast-1 \
            --name viva-cluster-${ENVIRONMENT}
        
        echo "✅ Deployment complete\!"
        echo "Run 'kubectl get nodes' to verify cluster access"
        ;;
    destroy)
        echo "⚠️  Destroying $ENVIRONMENT environment..."
        read -p "Are you sure? (yes/no): " confirm
        if [ "$confirm" == "yes" ]; then
            terraform destroy \
                -var-file="environments/${ENVIRONMENT}/terraform.tfvars" \
                -auto-approve
        else
            echo "Cancelled"
        fi
        ;;
esac
