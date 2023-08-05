{{/*
Expand the name of the chart.
*/}}
{{- define "nifcloud-cloud-controller-manager.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "nifcloud-cloud-controller-manager.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nifcloud-cloud-controller-manager.labels" -}}
helm.sh/chart: {{ include "nifcloud-cloud-controller-manager.chart" . }}
{{ include "nifcloud-cloud-controller-manager.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nifcloud-cloud-controller-manager.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nifcloud-cloud-controller-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
