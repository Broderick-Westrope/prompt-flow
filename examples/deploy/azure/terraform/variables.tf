variable "resource_group_name" {
  description = "Azure resource group name"
  type        = string
}

variable "location" {
  description = "Azure region for resource deployment"
  type        = string
  default     = "australiaeast"
}

variable "service_name" {
  description = "Name of the Container App service"
  type        = string
  default     = "pfctl"
}

variable "image_tag" {
  description = "Docker image tag to deploy"
  type        = string
  default     = "latest"
}

variable "min_replicas" {
  description = "Minimum number of Container App replicas (0 enables scale-to-zero)"
  type        = number
  default     = 0
}

variable "max_replicas" {
  description = "Maximum number of Container App replicas"
  type        = number
  default     = 10
}

variable "cpu" {
  description = "CPU cores per replica (valid: 0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 1.75, 2.0)"
  type        = number
  default     = 1.0
}

variable "memory" {
  description = "Memory per replica (must be at least 2x CPU for consumption plan, e.g. 2Gi for 1.0 CPU)"
  type        = string
  default     = "2Gi"
}

variable "concurrency" {
  description = "Maximum concurrent requests per replica"
  type        = number
  default     = 100
}

variable "execution_timeout" {
  description = "Maximum request timeout passed to pfctl serve --timeout"
  type        = string
  default     = "300s"
}

variable "openai_api_key" {
  description = "OpenAI API key (stored in Key Vault)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "anthropic_api_key" {
  description = "Anthropic API key (stored in Key Vault)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "azure_openai_endpoint" {
  description = "Azure OpenAI resource endpoint (e.g. https://your-resource.openai.azure.com/)"
  type        = string
  default     = ""
}

variable "azure_openai_api_key" {
  description = "Azure OpenAI API key (stored in Key Vault)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "flow_path" {
  description = "Path to the flow file inside the container"
  type        = string
  default     = "/flow.yaml"
}

variable "allow_external_ingress" {
  description = "Allow external (public) ingress to the Container App"
  type        = bool
  default     = false
}

variable "create_resource_group" {
  description = "Create a new resource group or use an existing one"
  type        = bool
  default     = true
}
