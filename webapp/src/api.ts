// TODO: типы генерируются из Go (internal/web/apidto), пока один тип описан вручную.
export type MeResponse = {
  id: number
  first_name: string
}

export async function fetchMe(initData: string): Promise<MeResponse> {
  const resp = await fetch('/api/me', { headers: { Authorization: `tma ${initData}` } })
  if (!resp.ok) throw new Error(`GET /api/me: ${resp.status}`)
  return (await resp.json()) satisfies MeResponse
}
