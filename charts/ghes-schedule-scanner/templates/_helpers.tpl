{{/*
Expand the name of the chart.
*/}}
{{- define "ghes-schedule-scanner.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "ghes-schedule-scanner.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Creates a string in the format "{chart-name}-{version}"
*/}}
{{- define "ghes-schedule-scanner.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Defines common labels used across Kubernetes resources
*/}}
{{- define "ghes-schedule-scanner.labels" -}}
helm.sh/chart: {{ include "ghes-schedule-scanner.chart" . }}
{{ include "ghes-schedule-scanner.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "ghes-schedule-scanner.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ghes-schedule-scanner.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Sets container image tag (uses .Chart.AppVersion if .Values.image.tag is not defined)
*/}}
{{- define "ghes-schedule-scanner.imageTag" -}}
{{- .Values.image.tag | default .Chart.AppVersion -}}
{{- end -}}

{{/*
configMap data helper
This helper template iterates through configMap.data values defined in values.yaml and generates configMap data entries.
*/}}
{{- define "ghes-schedule-scanner.configMapData" -}}
{{- range $key, $value := .Values.configMap.data }}
{{- if $value }}
{{ $key }}: {{ $value | quote }}
{{- end }}
{{- end }}
{{- end }}

{{/*
ConfigMap name helper
Returns the ConfigMap name to use (either external name or generated name)
*/}}
{{- define "ghes-schedule-scanner.configMapName" -}}
{{- if .Values.configMap.name -}}
{{- .Values.configMap.name -}}
{{- else -}}
{{- include "ghes-schedule-scanner.fullname" . -}}
{{- end -}}
{{- end -}}

{{/*
Exclude ConfigMap name helper
Returns the exclude ConfigMap name to use (either external name with -exclude suffix or generated name)
*/}}
{{- define "ghes-schedule-scanner.excludeConfigMapName" -}}
{{- if .Values.configMap.name -}}
{{- printf "%s-exclude" .Values.configMap.name -}}
{{- else -}}
{{- printf "%s-exclude" (include "ghes-schedule-scanner.fullname" .) -}}
{{- end -}}
{{- end -}}
