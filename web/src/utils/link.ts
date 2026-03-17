import type { Inbound, Client } from '@/api/inbound'

/**
 * Base64 编码（兼容 UTF-8）
 */
function base64Encode(str: string): string {
  return btoa(unescape(encodeURIComponent(str)))
}

/**
 * URL 安全的 Base64 编码
 */
function safeBase64Encode(str: string): string {
  return base64Encode(str)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

/**
 * 生成 VMess 链接（参考 x-ui 原版）
 */
export function generateVmessLink(
  host: string,
  port: number,
  uuid: string,
  alterId: number,
  network: string,
  security: string,
  remark: string,
  type?: string,
  wsHost?: string,
  wsPath?: string,
  grpcServiceName?: string,
  kcpSeed?: string
): string {
  let netType = 'none'
  let netHost = ''
  let netPath = ''

  if (network === 'tcp') {
    netType = type || 'none'
  } else if (network === 'kcp') {
    netType = type || 'none'
    netPath = kcpSeed || ''
  } else if (network === 'ws') {
    netPath = wsPath || ''
    netHost = wsHost || ''
  } else if (network === 'http' || network === 'h2') {
    network = 'h2'
    netPath = wsPath || ''
    netHost = wsHost || ''
  } else if (network === 'grpc') {
    netPath = grpcServiceName || ''
  }

  const obj = {
    v: '2',
    ps: remark,
    add: host,
    port: port,
    id: uuid,
    aid: alterId,
    net: network,
    type: netType,
    host: netHost,
    path: netPath,
    tls: security === 'tls' ? 'tls' : ''
  }

  // x-ui 原版使用格式化的 JSON
  return 'vmess://' + base64Encode(JSON.stringify(obj, null, 2))
}

/**
 * 生成 VLESS 链接（参考 x-ui 原版）
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
  wsHost?: string,
  grpcServiceName?: string,
  sni?: string,
  headerType?: string,
  kcpSeed?: string
): string {
  const params = new URLSearchParams()
  params.set('type', network)
  params.set('security', security || 'none')

  switch (network) {
    case 'tcp':
      if (headerType && headerType !== 'none') {
        params.set('headerType', headerType)
      }
      break
    case 'kcp':
      if (headerType) params.set('headerType', headerType)
      if (kcpSeed) params.set('seed', kcpSeed)
      break
    case 'ws':
      if (wsPath) params.set('path', wsPath)
      if (wsHost) params.set('host', wsHost)
      break
    case 'http':
    case 'h2':
      if (wsPath) params.set('path', wsPath)
      if (wsHost) params.set('host', wsHost)
      break
    case 'grpc':
      if (grpcServiceName) params.set('serviceName', grpcServiceName)
      break
  }

  if (security === 'tls' || security === 'xtls') {
    if (sni) params.set('sni', sni)
  }

  if (security === 'xtls' && flow) {
    params.set('flow', flow)
  }

  const url = new URL(`vless://${uuid}@${host}:${port}`)
  url.search = params.toString()
  url.hash = encodeURIComponent(remark)
  
  return url.toString()
}

/**
 * 生成 Trojan 链接（参考 x-ui 原版）
 */
export function generateTrojanLink(
  host: string,
  port: number,
  password: string,
  remark: string,
  sni?: string,
  network?: string,
  wsPath?: string,
  grpcServiceName?: string
): string {
  const params = new URLSearchParams()
  
  if (sni) params.set('sni', sni)
  if (network && network !== 'tcp') {
    params.set('type', network)
    if (network === 'ws' && wsPath) params.set('path', wsPath)
    if (network === 'grpc' && grpcServiceName) params.set('serviceName', grpcServiceName)
  }

  const query = params.toString()
  return `trojan://${encodeURIComponent(password)}@${host}:${port}${query ? '?' + query : ''}#${encodeURIComponent(remark)}`
}

/**
 * 生成 Shadowsocks 链接（参考 x-ui 原版）
 */
export function generateShadowsocksLink(
  host: string,
  port: number,
  method: string,
  password: string,
  remark: string
): string {
  // x-ui 使用 safeBase64 编码 method:password
  const userinfo = safeBase64Encode(`${method}:${password}`)
  return `ss://${userinfo}@${host}:${port}#${encodeURIComponent(remark)}`
}

/**
 * 解析入站规则的流设置
 */
function parseStreamSettings(streamSettingsStr: string) {
  try {
    const stream = JSON.parse(streamSettingsStr || '{}')
    return {
      network: stream.network || 'tcp',
      security: stream.security || 'none',
      wsPath: stream.wsSettings?.path,
      wsHost: stream.wsSettings?.headers?.Host,
      grpcServiceName: stream.grpcSettings?.serviceName,
      sni: stream.tlsSettings?.serverName || stream.xtlsSettings?.serverName,
      tcpHeaderType: stream.tcpSettings?.header?.type,
      kcpHeaderType: stream.kcpSettings?.header?.type,
      kcpSeed: stream.kcpSettings?.seed
    }
  } catch {
    return {
      network: 'tcp',
      security: 'none'
    }
  }
}

/**
 * 根据入站规则生成链接
 */
export function generateInboundLink(inbound: Inbound, host?: string): string {
  const targetHost = host || window.location.hostname

  try {
    const settings = JSON.parse(inbound.settings || '{}')
    const stream = parseStreamSettings(inbound.streamSettings)

    switch (inbound.protocol) {
      case 'vmess': {
        const client = settings.clients?.[0]
        if (!client) return ''
        return generateVmessLink(
          targetHost,
          inbound.port,
          client.id,
          client.alterId || 0,
          stream.network,
          stream.security,
          inbound.remark || `${inbound.port}`,
          stream.tcpHeaderType,
          stream.wsHost,
          stream.wsPath,
          stream.grpcServiceName,
          stream.kcpSeed
        )
      }

      case 'vless': {
        const client = settings.clients?.[0]
        if (!client) return ''
        return generateVlessLink(
          targetHost,
          inbound.port,
          client.id,
          stream.network,
          stream.security,
          inbound.remark || `${inbound.port}`,
          client.flow,
          stream.wsPath,
          stream.wsHost,
          stream.grpcServiceName,
          stream.sni,
          stream.tcpHeaderType || stream.kcpHeaderType,
          stream.kcpSeed
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
          stream.sni,
          stream.network,
          stream.wsPath,
          stream.grpcServiceName
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
  const stream = parseStreamSettings(inbound.streamSettings)
  const remark = client.remark || inbound.remark || `${inbound.port}`

  try {
    switch (inbound.protocol) {
      case 'vmess':
        return generateVmessLink(
          targetHost,
          inbound.port,
          client.uuid,
          0,
          stream.network,
          stream.security,
          remark,
          stream.tcpHeaderType,
          stream.wsHost,
          stream.wsPath,
          stream.grpcServiceName,
          stream.kcpSeed
        )

      case 'vless':
        return generateVlessLink(
          targetHost,
          inbound.port,
          client.uuid,
          stream.network,
          stream.security,
          remark,
          client.flow,
          stream.wsPath,
          stream.wsHost,
          stream.grpcServiceName,
          stream.sni,
          stream.tcpHeaderType || stream.kcpHeaderType,
          stream.kcpSeed
        )

      case 'trojan':
        return generateTrojanLink(
          targetHost,
          inbound.port,
          client.password,
          remark,
          stream.sni,
          stream.network,
          stream.wsPath,
          stream.grpcServiceName
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

  return base64Encode(links.join('\n'))
}
