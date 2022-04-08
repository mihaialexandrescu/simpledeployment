{{/*
Expand the name of the chart.
*/}}
{{- define "simpledeployment-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Provides the namespace the chart will be installed in using the builtin .Release.Namespace.
*/}}
{{- define "simpledeployment-operator.namespace" -}}
{{ .Release.Namespace }}
{{- end -}}


{{/*
Overrideable version for container image tags.
*/}}
{{- define "simpledeployment-operator.version" -}}
{{- .Values.image.tag | default (printf "%s" .Chart.AppVersion) -}}
{{- end -}}


{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "simpledeployment-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "simpledeployment-operator.labels" -}}
helm.sh/chart: {{ include "simpledeployment-operator.chart" . }}
{{ include "simpledeployment-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "simpledeployment-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "simpledeployment-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
