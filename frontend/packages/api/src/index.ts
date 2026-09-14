import axios, { type AxiosRequestConfig } from 'axios'
import type { ApiResponse, PageQuery, PageResult, UserPageListItem } from '@ark-go-starter/types'

export const http = axios.create({
  baseURL: '/v1',
  timeout: 10000,
})

http.interceptors.response.use(
  (response) => {
    const body = response.data as ApiResponse
    if (body.code !== 0) {
      return Promise.reject(new Error(body.msg || `request failed, code=${body.code}`))
    }
    return body.data as unknown as typeof response
  },
  (error) => Promise.reject(error),
)

export async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const response = await http.request<unknown, unknown>(config)
  return response as T
}

export function getUsers(params: PageQuery = {}) {
  return request<PageResult<UserPageListItem>>({
    url: '/demo/users',
    method: 'get',
    params,
  })
}
