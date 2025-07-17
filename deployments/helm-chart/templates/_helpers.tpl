{{/*
Expand the name of the chart.
*/}}
{{- define "mimir-insights.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "mimir-insights.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}[object Object]{- .Release.Name | trunc 63trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63trimSuffix - }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define mimir-insights.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace +  | trunc 63trimSuffix "-" }}
{{- end }}

[object Object]{/*
Common labels
*/}}
{{- definemimir-insights.labels" -}}
helm.sh/chart: {{ include mimir-insights.chart" . }}
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
{{- defaultdefault" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the namespace name
*/}}
{{- definemimir-insights.namespace" -}}
{{- if .Values.namespace.create }}
{{- .Values.namespace.name }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "mimir-insights.image" -}}[object Object][object Object]- printf %s/%s:%s" .Values.global.imageRegistry .Values.backend.image.repository .Values.backend.image.tag }}
{{- end }}

{{/*
Create the frontend image name
*/}}
{{- define "mimir-insights.frontendImage" -}}[object Object][object Object]- printf %s/%s:%s" .Values.global.imageRegistry .Values.frontend.image.repository .Values.frontend.image.tag }}
{{- end }} 