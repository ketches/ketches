{{- if .Values.postgres.enabled }}
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ include "ketches.postgres.fullname" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
    app.kubernetes.io/component: postgres
spec:
  serviceName: {{ include "ketches.postgres.fullname" . | quote }}
  replicas: 1
  selector:
    matchLabels:
      {{- include "ketches.componentSelectorLabels" (dict "context" . "component" "postgres") | nindent 6 }}
  template:
    metadata:
      annotations:
        checksum/secret: {{ include (print $.Template.BasePath "/secret.yaml.tpl") . | sha256sum | quote }}
        {{- with .Values.postgres.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      labels:
        {{- include "ketches.componentSelectorLabels" (dict "context" . "component" "postgres") | nindent 8 }}
        {{- with .Values.postgres.podLabels }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      containers:
        - name: postgres
          image: "{{ .Values.postgres.image.repository }}:{{ .Values.postgres.image.tag }}"
          imagePullPolicy: {{ .Values.postgres.image.pullPolicy }}
          ports:
            - name: postgres
              containerPort: {{ .Values.postgres.service.port }}
              protocol: TCP
          env:
            - name: POSTGRES_DB
              value: {{ .Values.postgres.auth.database | quote }}
            - name: POSTGRES_USER
              value: {{ .Values.postgres.auth.username | quote }}
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "ketches.secretName" . | quote }}
                  key: postgres-password
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
          resources:
            {{- toYaml .Values.postgres.resources | nindent 12 }}
      {{- if not .Values.postgres.persistence.enabled }}
      volumes:
        - name: data
          emptyDir: {}
      {{- end }}
      {{- with .Values.postgres.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.postgres.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.postgres.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
  {{- if .Values.postgres.persistence.enabled }}
  volumeClaimTemplates:
    - metadata:
        name: data
        labels:
          {{- include "ketches.labels" . | nindent 10 }}
          app.kubernetes.io/component: postgres
      spec:
        accessModes:
          {{- toYaml .Values.postgres.persistence.accessModes | nindent 10 }}
        resources:
          requests:
            storage: {{ .Values.postgres.persistence.size }}
        {{- if .Values.postgres.persistence.storageClass }}
        storageClassName: {{ .Values.postgres.persistence.storageClass | quote }}
        {{- end }}
  {{- end }}
{{- end }}
