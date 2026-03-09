// Standardized pagination types

export interface PaginationParams {
  page?: number
  page_size?: number
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
