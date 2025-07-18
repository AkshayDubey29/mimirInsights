{{/*
Expand the name of the chart.
*/}}
{{- define "mimir-insights.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "mimir-insights.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "mimir-insights.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "mimir-insights.labels" -}}
helm.sh/chart: {{ include "mimir-insights.chart" . }}
{{ include "mimir-insights.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "mimir-insights.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mimir-insights.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "mimir-insights.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "mimir-insights.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the namespace name
*/}}
{{- define "mimir-insights.namespace" -}}
{{- if .Values.namespace.create }}
{{- .Values.namespace.name }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "mimir-insights.image" -}}
{{- printf "%s/%s:%s" .Values.imageRegistry .Values.backend.image.repository .Values.backend.image.tag }}
{{- end }}

{{/*
Create the frontend image name
*/}}
{{- define "mimir-insights.frontendImage" -}}
{{- printf "%s/%s:%s" .Values.imageRegistry .Values.frontend.image.repository .Values.frontend.image.tag }}
{{- end }} 