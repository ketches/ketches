// Standardized pagination types

export interface PaginationParams {
  page?: number
  pageSize?: number
  search?: string
  group_id?: string
  favorite?: boolean
}

export interface PaginationResponse {
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface SimpleResponse {
  id: string
  slug: string
  name: string
  description?: string
  status?: string
  metadata?: Record<string, string>
}
