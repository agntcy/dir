{{/*
Expand the name of the chart.
*/}}
{{- define "validation.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "validation.fullname" -}}
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
{{- define "validation.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "validation.labels" -}}
helm.sh/chart: {{ include "validation.chart" . }}
{{ include "validation.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "validation.selectorLabels" -}}
app.kubernetes.io/name: {{ include "validation.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
HTTP component labels
*/}}
{{- define "validation.http.labels" -}}
{{ include "validation.labels" . }}
app.kubernetes.io/component: http
{{- end }}

{{/*
HTTP selector labels
*/}}
{{- define "validation.http.selectorLabels" -}}
{{ include "validation.selectorLabels" . }}
app.kubernetes.io/component: http
{{- end }}

{{/*
DNS component labels
*/}}
{{- define "validation.dns.labels" -}}
{{ include "validation.labels" . }}
app.kubernetes.io/component: dns
{{- end }}

{{/*
DNS selector labels
*/}}
{{- define "validation.dns.selectorLabels" -}}
{{ include "validation.selectorLabels" . }}
app.kubernetes.io/component: dns
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "validation.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "validation.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Validate that at least one component is enabled
*/}}
{{- define "validation.validateEnabled" -}}
{{- if and (not .Values.http.enabled) (not .Values.dns.enabled) }}
{{- fail "At least one component (http or dns) must be enabled" }}
{{- end }}
{{- end }}
