{{/*
Expand the name of the chart.
*/}}
{{- define "chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "chart.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "chart.labels" -}}
helm.sh/chart: {{ include "chart.chart" . }}
{{ include "chart.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "chart.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "chart.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate cloud provider-specific annotations for routing service
*/}}
{{- define "chart.routingService.annotations" -}}
{{- $annotations := dict -}}
{{- if and .Values.routingService .Values.routingService.cloudProvider -}}
{{- if eq .Values.routingService.cloudProvider "aws" -}}
{{- $_ := set $annotations "service.beta.kubernetes.io/aws-load-balancer-type" "nlb" -}}
{{- $_ := set $annotations "service.beta.kubernetes.io/aws-load-balancer-scheme" "internet-facing" -}}
{{- $_ := set $annotations "service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled" "true" -}}
{{- if .Values.routingService.aws.nlbTargetType -}}
{{- $_ := set $annotations "service.beta.kubernetes.io/aws-load-balancer-nlb-target-type" .Values.routingService.aws.nlbTargetType -}}
{{- end -}}
{{- if .Values.routingService.aws.internal -}}
{{- $_ := set $annotations "service.beta.kubernetes.io/aws-load-balancer-internal" "true" -}}
{{- $_ := set $annotations "service.beta.kubernetes.io/aws-load-balancer-scheme" "internal" -}}
{{- end -}}
{{- else if eq .Values.routingService.cloudProvider "gcp" -}}
{{- $_ := set $annotations "cloud.google.com/load-balancer-type" "External" -}}
{{- if .Values.routingService.gcp.internal -}}
{{- $_ := set $annotations "cloud.google.com/load-balancer-type" "Internal" -}}
{{- end -}}
{{- if .Values.routingService.gcp.backendConfig -}}
{{- $_ := set $annotations "cloud.google.com/backend-config" .Values.routingService.gcp.backendConfig -}}
{{- end -}}
{{- else if eq .Values.routingService.cloudProvider "azure" -}}
{{- $_ := set $annotations "service.beta.kubernetes.io/azure-load-balancer-internal" "false" -}}
{{- if .Values.routingService.azure.internal -}}
{{- $_ := set $annotations "service.beta.kubernetes.io/azure-load-balancer-internal" "true" -}}
{{- end -}}
{{- if .Values.routingService.azure.resourceGroup -}}
{{- $_ := set $annotations "service.beta.kubernetes.io/azure-load-balancer-resource-group" .Values.routingService.azure.resourceGroup -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- /* Merge provider annotations with custom annotations (custom takes precedence) */ -}}
{{- if and .Values.routingService .Values.routingService.annotations -}}
{{- range $key, $value := .Values.routingService.annotations -}}
{{- $_ := set $annotations $key $value -}}
{{- end -}}
{{- end -}}
{{- if $annotations -}}
{{- toYaml $annotations -}}
{{- end -}}
{{- end -}}
