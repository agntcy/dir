{{/*
Copyright AGNTCY Contributors (https://github.com/agntcy)
SPDX-License-Identifier: Apache-2.0
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "envoy-authz.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "envoy-authz.fullname" -}}
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
{{- define "envoy-authz.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "envoy-authz.labels" -}}
helm.sh/chart: {{ include "envoy-authz.chart" . }}
{{ include "envoy-authz.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "envoy-authz.selectorLabels" -}}
app.kubernetes.io/name: {{ include "envoy-authz.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Envoy gateway labels
*/}}
{{- define "envoy-authz.envoyLabels" -}}
{{ include "envoy-authz.labels" . }}
app.kubernetes.io/component: envoy-gateway
{{- end }}

{{/*
Envoy gateway selector labels
*/}}
{{- define "envoy-authz.envoySelector" -}}
{{ include "envoy-authz.selectorLabels" . }}
app.kubernetes.io/component: envoy-gateway
{{- end }}

{{/*
AuthZ server labels
*/}}
{{- define "envoy-authz.authzLabels" -}}
{{ include "envoy-authz.labels" . }}
app.kubernetes.io/component: authz-server
{{- end }}

{{/*
AuthZ server selector labels
*/}}
{{- define "envoy-authz.authzSelector" -}}
{{ include "envoy-authz.selectorLabels" . }}
app.kubernetes.io/component: authz-server
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "envoy-authz.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "envoy-authz.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Envoy service account name (for SPIFFE ID)
*/}}
{{- define "envoy-authz.envoyServiceAccountName" -}}
{{- printf "%s-envoy-gateway" (include "envoy-authz.fullname" .) }}
{{- end }}
