export interface ApiResponse<T = unknown> {
  code: number
  requestID: string
  data: T
  msg: string
}

export interface PageQuery {
  page?: number
  pageSize?: number
}

export interface PageResult<T> {
  list: T[]
  total: number
}

export interface UserPageListItem {
  userID: number
  companyID?: number
  departmentID?: number
  name?: string
}
