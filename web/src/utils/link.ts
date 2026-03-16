import type { Inbound, Client } from '@/api/inbound'

/**
 * 生成 VMess 链接
 */
export function generateVmessLink(
  host: string,
  port: number,
  uuid: string,
  alterId: number,
  network: string,
  security: string,
  remark: string,
  wsPath?: string,
  wsHost?: string,
  grpcServiceName?: string
): string {
  const config = {
    v: '2',
    ps: remark,
    add: host,
    port: port,
    id: uuid,
    aid: alterId,
    net: network,
    type: 'none',
    host: wsHost || '',
    path: wsPath || '',
    tls: security === 'tls' ? 'tls' : ''
  }

  if (network === 'grpc') {
    config.path = grpcServiceName || ''
  }

  return 'vmess://' + btoa(JSON.stringify(config))
}

/**
 * 生成 VLESS 链接
 */
export function generateVlessLink(
  host: string,
  port: number,
  uuid: string,
  network: string,
  security: string,
  remark: string,
  flow?: string,
  wsPath?: string,
  grpcServiceName?: string,
  sni?: string
): string {
  const params = new URLSearchParams()
  params.set('type', network)

  if (security) params.set('security', security)
  if (flow) params.set('flow', flow)
  if (wsPath && network === 'ws') params.set('path', wsPath)
  if (grpcServiceName && network === 'grpc') params.set('serviceName', grpcServiceName)
  if (sni && security === 'tls') params.set('sni', sni)

  return `vless://${uuid}@${host}:${port}?${params.toString()}#${encodeURIComponent(remark)}`
}

/**
 * 生成 Trojan 链接
 */
export function generateTrojanLink(
  host: string,
  port: number,
  password: string,
  remark: string,
  sni?: string
): string {
  const params = new URLSearchParams()
  if (sni) params.set('sni', sni)

  const query = params.toString()
  return `trojan://${password}@${host}:${port}${query ? '?' + query : ''}#${encodeURIComponent(remark)}`
}

/**
 * 生成 Shadowsocks 链接
 */
export function generateShadowsocksLink(
  host: string,
  port: number,
  method: string,
  password: string,
  remark: string
): string {
  const userinfo = btoa(`${method}:${password}`)
  return `ss://${userinfo}@${host}:${port}#${encodeURIComponent(remark)}`
}

/**
 * 根据入站规则生成链接
 */
export function generateInboundLink(inbound: Inbound, host?: string): string {
  const targetHost = host || window.location.hostname

  try {
    const settings = JSON.parse(inbound.settings || '{}')
    const streamSettings = JSON.parse(inbound.streamSettings || '{}')

    const network = streamSettings.network || 'tcp'
    const security = streamSettings.security || 'none'
    const wsPath = streamSettings.wsSettings?.path
    const wsHost = streamSettings.wsSettings?.headers?.Host
    const grpcServiceName = streamSettings.grpcSettings?.serviceName
    const sni = streamSettings.tlsSettings?.serverName

    switch (inbound.protocol) {
      case 'vmess': {
        const client = settings.clients?.[0]
        if (!client) return ''
        return generateVmessLink(
          targetHost,
          inbound.port,
          client.id,
          client.alterId || 0,
          network,
          security,
          inbound.remark || `${inbound.port}`,
          wsPath,
          wsHost,
          grpcServiceName
        )
      }

      case 'vless': {
        const client = settings.clients?.[0]
        if (!client) return ''
        return generateVlessLink(
          targetHost,
          inbound.port,
          client.id,
          network,
          security,
          inbound.remark || `${inbound.port}`,
          client.flow,
          wsPath,
          grpcServiceName,
          sni
        )
      }

      case 'trojan': {
        const client = settings.clients?.[0]
        if (!client) return ''
        return generateTrojanLink(
          targetHost,
          inbound.port,
          client.password,
          inbound.remark || `${inbound.port}`,
          sni
        )
      }

      case 'shadowsocks': {
        return generateShadowsocksLink(
          targetHost,
          inbound.port,
          settings.method || 'aes-256-gcm',
          settings.password || '',
          inbound.remark || `${inbound.port}`
        )
      }

      default:
        return ''
    }
  } catch (e) {
    console.error('生成链接失败:', e)
    return ''
  }
}

/**
 * 根据客户端生成链接
 */
export function generateClientLink(
  client: Client,
  inbound: Inbound,
  host?: string
): string {
  const targetHost = host || window.location.hostname

  try {
    const streamSettings = JSON.parse(inbound.streamSettings || '{}')
    const network = streamSettings.network || 'tcp'
    const security = streamSettings.security || 'none'
    const wsPath = streamSettings.wsSettings?.path
    const wsHost = streamSettings.wsSettings?.headers?.Host
    const grpcServiceName = streamSettings.grpcSettings?.serviceName
    const sni = streamSettings.tlsSettings?.serverName

    switch (inbound.protocol) {
      case 'vmess':
        return generateVmessLink(
          targetHost,
          inbound.port,
          client.uuid,
          0,
          network,
          security,
          client.email || 'client',
          wsPath,
          wsHost,
          grpcServiceName
        )

      case 'vless':
        return generateVlessLink(
          targetHost,
          inbound.port,
          client.uuid,
          network,
          security,
          client.email || 'client',
          client.flow,
          wsPath,
          grpcServiceName,
          sni
        )

      case 'trojan':
        return generateTrojanLink(
          targetHost,
          inbound.port,
          client.password,
          client.email || 'client',
          sni
        )

      default:
        return ''
    }
  } catch (e) {
    console.error('生成客户端链接失败:', e)
    return ''
  }
}

/**
 * 生成订阅内容（Base64 编码的链接列表）
 */
export function generateSubscription(inbounds: Inbound[], host?: string): string {
  const links = inbounds
    .filter(i => i.enable)
    .map(i => generateInboundLink(i, host))
    .filter(link => link)

  return btoa(links.join('\n'))
}
