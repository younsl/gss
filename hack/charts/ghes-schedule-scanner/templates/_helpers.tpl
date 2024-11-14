{{/* 
Creates a string in the format "{chart-name}-{version}"
*/}}
{{- define "ghes-schedule-scanner.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version -}}
{{- end -}}

{{/* 
Defines common labels used across Kubernetes resources
*/}}
{{- define "ghes-schedule-scanner.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ printf "%s" .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* 
Generates resource name in format "{chart-name}-{release-name}"
*/}}
{{- define "ghes-schedule-scanner.name" -}}
{{- printf "%s-%s" .Chart.Name .Release.Name -}}
{{- end -}}

{{/* 
Sets container image tag (uses .Chart.AppVersion if .Values.image.tag is not defined)
*/}}
{{- define "ghes-schedule-scanner.imageTag" -}}
{{- .Values.image.tag | default .Chart.AppVersion -}}
{{- end -}}