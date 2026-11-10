{{/*
Common labels
*/}}
{{- define "cloudchamber.labels" -}}
app: cloudchamber
app.kubernetes.io/name: cloudchamber
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cloudchamber.selectorLabels" -}}
app: cloudchamber
app.kubernetes.io/name: cloudchamber
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Database environment variables for CloudChamber
*/}}
{{- define "cloudchamber.dbEnvVars" -}}
- name: CLOUDCHAMBER_DB_USER
  valueFrom:
    configMapKeyRef:
      name: cloudchamber-config
      key: db_user
- name: CLOUDCHAMBER_DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: cloudchamber-secret
      key: db_password
- name: CLOUDCHAMBER_DB_HOST_PORT
  valueFrom:
    configMapKeyRef:
      name: cloudchamber-config
      key: db_hostport
- name: CLOUDCHAMBER_DB_DISABLE_TLS
  valueFrom:
    configMapKeyRef:
      name: cloudchamber-config
      key: db_disabletls
{{- end -}}

{{/*
Kubernetes metadata environment variables
*/}}
{{- define "cloudchamber.k8sEnvVars" -}}
- name: KUBERNETES_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: KUBERNETES_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: KUBERNETES_POD_IP
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
- name: KUBERNETES_NODE_NAME
  valueFrom:
    fieldRef:
      fieldPath: spec.nodeName
{{- end -}}
