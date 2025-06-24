export interface QueryAndPagedRequest {
    pageNo: number;
    pageSize: number;
    query?: string;
    sortBy?: string;
    sortOrder?: 'ASC' | 'DESC';
}