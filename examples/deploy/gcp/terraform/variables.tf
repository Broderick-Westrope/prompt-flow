variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for resource deployment"
  type        = string
  default     = "australia-southeast1"
}

variable "service_name" {
  description = "Name of the Cloud Run service"
  type        = string
  default     = "pfctl"
}

variable "image_tag" {
  description = "Docker image tag to deploy"
  type        = string
  default     = "latest"
}

variable "min_instances" {
  description = "Minimum number of Cloud Run instances"
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances"
  type        = number
  default     = 10
}

variable "concurrency" {
  description = "Maximum concurrent requests per instance"
  type        = number
  default     = 100
}

variable "cpu" {
  description = "CPU allocation for each instance"
  type        = string
  default     = "1"
}

variable "memory" {
  description = "Memory allocation for each instance"
  type        = string
  default     = "512Mi"
}

variable "execution_timeout" {
  description = "Maximum request timeout"
  type        = string
  default     = "300s"
}

variable "openai_api_key" {
  description = "OpenAI API key (stored in Secret Manager)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "anthropic_api_key" {
  description = "Anthropic API key (stored in Secret Manager)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "allow_unauthenticated" {
  description = "Allow unauthenticated access to the Cloud Run service"
  type        = bool
  default     = false
}
