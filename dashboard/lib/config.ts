export const config = {
  apiGatewayUrl:
    process.env.NEXT_PUBLIC_API_GATEWAY_URL || "http://localhost:8080",
  wsUrl: process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080/ws",
};

export const getApiUrl = (path: string) => {
  return `${config.apiGatewayUrl}${path}`;
};
