apiVersion: v1
kind: Service
metadata:
  name: {{ include "ketches.ui.fullname" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
    app.kubernetes.io/component: ui
spec:
  type: {{ .Values.ui.service.type }}
  ports:
    - name: http
      port: {{ .Values.ui.service.port }}
      targetPort: http
      protocol: TCP
      {{- if and (eq .Values.ui.service.type "NodePort") (.Values.ui.service.nodePort) }}
      nodePort: {{ .Values.ui.service.nodePort }}
      {{- end }}
  selector:
    {{- include "ketches.componentSelectorLabels" (dict "context" . "component" "ui") | nindent 4 }}
