import client from './client'
import { authenticatedFetch } from '@/lib/auth-session'

export interface FileInfo {
  name: string
  type: 'file' | 'dir' | 'link'
  size: number
  modTime: number
  permissions: string
}

export interface ListFilesResponse {
  path: string
  files: FileInfo[]
}

export interface ReadFileResponse {
  path: string
  content: string
  size: number
}

export const fileExplorerApi = {
  getHomeDir: async (appId: string, instanceName: string, container: string) => {
    return client.get(`/v1/apps/${appId}/instances/${instanceName}/files/home`, {
      params: { container },
    }) as Promise<{ path: string }>
  },

  listFiles: async (appId: string, instanceName: string, container: string, path: string = '/') => {
    return client.get(`/v1/apps/${appId}/instances/${instanceName}/files`, {
      params: { container, path },
    }) as Promise<ListFilesResponse>
  },

  readFile: async (appId: string, instanceName: string, container: string, path: string) => {
    return client.get(`/v1/apps/${appId}/instances/${instanceName}/files/read`, {
      params: { container, path },
    }) as Promise<ReadFileResponse>
  },

  writeFile: async (appId: string, instanceName: string, container: string, path: string, content: string) => {
    return client.post(`/v1/apps/${appId}/instances/${instanceName}/files/write`, { path, content }, {
      params: { container },
    })
  },

  mkdir: async (appId: string, instanceName: string, container: string, path: string) => {
    return client.post(`/v1/apps/${appId}/instances/${instanceName}/files/mkdir`, { path }, {
      params: { container },
    })
  },

  deleteFile: async (appId: string, instanceName: string, container: string, path: string) => {
    return client.post(`/v1/apps/${appId}/instances/${instanceName}/files/delete`, { path }, {
      params: { container },
    })
  },

  moveFile: async (appId: string, instanceName: string, container: string, source: string, destination: string) => {
    return client.post(`/v1/apps/${appId}/instances/${instanceName}/files/move`, { source, destination }, {
      params: { container },
    })
  },

  copyFile: async (appId: string, instanceName: string, container: string, source: string, destination: string) => {
    return client.post(`/v1/apps/${appId}/instances/${instanceName}/files/copy`, { source, destination }, {
      params: { container },
    })
  },

  downloadFile: async (appId: string, instanceName: string, container: string, path: string) => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    const params = new URLSearchParams({ container, path })

    const response = await authenticatedFetch(`${baseUrl}/v1/apps/${appId}/instances/${instanceName}/files/download?${params}`)

    if (!response.ok) {
      throw new Error(`Download failed: ${response.statusText}`)
    }

    const blob = await response.blob()
    const filename = path.split('/').pop() || 'download'

    // Trigger browser download
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  },

  downloadDir: async (appId: string, instanceName: string, container: string, path: string) => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    const params = new URLSearchParams({ container, path })

    const response = await authenticatedFetch(`${baseUrl}/v1/apps/${appId}/instances/${instanceName}/files/download-dir?${params}`)

    if (!response.ok) {
      throw new Error(`Download failed: ${response.statusText}`)
    }

    const blob = await response.blob()
    const dirname = path.split('/').pop() || 'download'

    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${dirname}.tar`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  },

  uploadFile: async (appId: string, instanceName: string, container: string, destDir: string, file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return client.post(`/v1/apps/${appId}/instances/${instanceName}/files/upload`, formData, {
      params: { container, path: destDir },
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000, // 2 minutes for uploads
    })
  },

  compressFiles: async (appId: string, instanceName: string, container: string, baseDir: string, fileNames: string[], destPath: string) => {
    return client.post(`/v1/apps/${appId}/instances/${instanceName}/files/compress`, {
      baseDir, fileNames, destPath,
    }, { params: { container } })
  },

  compressAndDownload: async (appId: string, instanceName: string, container: string, baseDir: string, fileNames: string[], archiveName: string) => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'

    const response = await authenticatedFetch(`${baseUrl}/v1/apps/${appId}/instances/${instanceName}/files/compress-download?container=${encodeURIComponent(container)}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ baseDir, fileNames, archiveName }),
    })

    if (!response.ok) {
      throw new Error(`Compress & download failed: ${response.statusText}`)
    }

    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = archiveName || 'archive.tar.gz'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  },
}
