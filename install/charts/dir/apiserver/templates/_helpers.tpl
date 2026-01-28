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

{{/*
PostgreSQL credentials helpers.
Single source of truth based on configuration mode:
  - postgresql.enabled=true  → use postgresql.auth.* (subchart manages credentials)
  - externalSecrets.enabled  → credentials from Vault (no values needed here)
  - otherwise                → use secrets.postgresAuth.* (explicit credentials)
*/}}

{{/*
Get PostgreSQL username.
*/}}
{{- define "chart.postgres.username" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.auth.username | default "dir" -}}
{{- else -}}
{{- .Values.secrets.postgresAuth.username | default "dir" -}}
{{- end -}}
{{- end -}}

{{/*
Get PostgreSQL password.
*/}}
{{- define "chart.postgres.password" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.auth.password | default (randAlphaNum 32) -}}
{{- else -}}
{{- .Values.secrets.postgresAuth.password | default (randAlphaNum 32) -}}
{{- end -}}
{{- end -}}

{{/*
Get PostgreSQL database name.
*/}}
{{- define "chart.postgres.database" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.auth.database | default "dir" -}}
{{- else -}}
{{- .Values.database.postgres.database | default "dir" -}}
{{- end -}}
{{- end -}}

{{/*
Check if postgres credentials are configured.
Returns true when credentials will be available (from any source).
*/}}
{{- define "chart.postgres.credentialsConfigured" -}}
{{- if or .Values.postgresql.enabled (and .Values.externalSecrets.enabled .Values.externalSecrets.postgresAuth.enabled) .Values.secrets.postgresAuth.username .Values.secrets.postgresAuth.password -}}
true
{{- end -}}
{{- end -}}

{{/*
Create a name for the reconciler deployment.
Uses release name + "reconciler" (without the chart name "apiserver").
*/}}
{{- define "chart.reconcilerName" -}}
{{- printf "%s-reconciler" .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/*
Get OCI registry address.
When zot.enabled=true (internal Zot), returns the Kubernetes internal service address.
Otherwise, returns the user-configured config.store.oci.registry_address.

Note: Uses (get .Values.zot "enabled") for nil-safe access since zot may not be defined.
*/}}
{{- define "chart.oci.registryAddress" -}}
{{- if and .Values.zot (get .Values.zot "enabled") -}}
{{- printf "%s-zot.%s.svc.cluster.local:5000" .Release.Name .Release.Namespace -}}
{{- else -}}
{{- .Values.config.store.oci.registry_address -}}
{{- end -}}
{{- end -}}

{{/*
Get OCI repository name.
Returns user-configured value, or "dir" as default when using internal Zot.

Note: Uses (get .Values.zot "enabled") for nil-safe access since zot may not be defined.
*/}}
{{- define "chart.oci.repositoryName" -}}
{{- if .Values.config.store.oci.repository_name -}}
{{- .Values.config.store.oci.repository_name -}}
{{- else if and .Values.zot (get .Values.zot "enabled") -}}
dir
{{- end -}}
{{- end -}}
