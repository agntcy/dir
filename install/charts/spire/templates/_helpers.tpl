{{/*
Expand the name of the chart.
*/}}
{{- define "spire.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "spire.fullname" -}}
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
{{- define "spire.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Namespace name
*/}}
{{- define "spire.namespace" -}}
{{- .Release.Namespace }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "spire.labels" -}}
helm.sh/chart: {{ include "spire.chart" . }}
{{ include "spire.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "spire.selectorLabels" -}}
app.kubernetes.io/name: {{ include "spire.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
SPIRE Server labels
*/}}
{{- define "spire.server.labels" -}}
{{ include "spire.labels" . }}
app.kubernetes.io/component: server
{{- end }}

{{/*
SPIRE Server selector labels
*/}}
{{- define "spire.server.selectorLabels" -}}
{{ include "spire.selectorLabels" . }}
app.kubernetes.io/component: server
{{- end }}

{{/*
SPIRE Agent labels
*/}}
{{- define "spire.agent.labels" -}}
{{ include "spire.labels" . }}
app.kubernetes.io/component: agent
{{- end }}

{{/*
SPIRE Agent selector labels
*/}}
{{- define "spire.agent.selectorLabels" -}}
{{ include "spire.selectorLabels" . }}
app.kubernetes.io/component: agent
{{- end }}

{{/*
SPIRE Server service account name
*/}}
{{- define "spire.server.serviceAccountName" -}}
{{- if .Values.server.serviceAccount.create }}
{{- default (printf "%s-server" (include "spire.fullname" .)) .Values.server.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.server.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
SPIRE Agent service account name
*/}}
{{- define "spire.agent.serviceAccountName" -}}
{{- if .Values.agent.serviceAccount.create }}
{{- default (printf "%s-agent" (include "spire.fullname" .)) .Values.agent.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.agent.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
SPIRE Server full name
*/}}
{{- define "spire.server.fullname" -}}
{{- printf "%s-server" (include "spire.fullname" .) }}
{{- end }}

{{/*
SPIRE Agent full name
*/}}
{{- define "spire.agent.fullname" -}}
{{- printf "%s-agent" (include "spire.fullname" .) }}
{{- end }}

{{/*
Trust domain
*/}}
{{- define "spire.trustDomain" -}}
{{- .Values.global.spire.trustDomain | default .Values.agent.config.trustDomain | default .Values.server.config.trustDomain | default "example.org" }}
{{- end }}

{{/*
Cluster name
*/}}
{{- define "spire.clusterName" -}}
{{- .Values.global.spire.clusterName | default "demo-cluster" }}
{{- end }}

{{/*
Common annotations
*/}}
{{- define "spire.annotations" -}}
{{- with .Values.commonAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}
