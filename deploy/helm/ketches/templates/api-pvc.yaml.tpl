{{- if .Values.api.persistence.enabled }}
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ printf "%s-build-logs" (include "ketches.api.fullname" .) | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
    app.kubernetes.io/component: api
spec:
  accessModes:
    {{- toYaml .Values.api.persistence.accessModes | nindent 4 }}
  resources:
    requests:
      storage: {{ .Values.api.persistence.size }}
  {{- if .Values.api.persistence.storageClass }}
  storageClassName: {{ .Values.api.persistence.storageClass | quote }}
  {{- end }}
{{- end }}
