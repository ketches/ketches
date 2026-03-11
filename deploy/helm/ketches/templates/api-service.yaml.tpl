apiVersion: v1
kind: Service
metadata:
  name: {{ include "ketches.api.fullname" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
    app.kubernetes.io/component: api
spec:
  type: {{ .Values.api.service.type }}
  ports:
    - name: http
      port: {{ .Values.api.service.port }}
      targetPort: http
      protocol: TCP
  selector:
    {{- include "ketches.componentSelectorLabels" (dict "context" . "component" "api") | nindent 4 }}
