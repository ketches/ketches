{{- if .Values.postgres.enabled }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "ketches.postgres.fullname" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
    app.kubernetes.io/component: postgres
spec:
  clusterIP: None
  ports:
    - name: postgres
      port: {{ .Values.postgres.service.port }}
      targetPort: postgres
      protocol: TCP
  selector:
    {{- include "ketches.componentSelectorLabels" (dict "context" . "component" "postgres") | nindent 4 }}
{{- end }}
