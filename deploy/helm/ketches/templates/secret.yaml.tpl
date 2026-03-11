apiVersion: v1
kind: Secret
metadata:
  name: {{ include "ketches.secretName" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
type: Opaque
stringData:
  jwt-secret: {{ required "values.config.jwtSecret is required" .Values.config.jwtSecret | quote }}
  db-source: {{ include "ketches.database.source" . | quote }}
  {{- if .Values.postgres.enabled }}
  postgres-password: {{ .Values.postgres.auth.password | quote }}
  {{- end }}
