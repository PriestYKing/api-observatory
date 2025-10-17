"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Alert } from "@/components/ui/alert";
import RealTimeMetrics from "../components/RealTimeMetrics";
import CostDashboard from "../components/CostDashboard";
import OptimizationPanel from "../components/OptimizationPanel";
import { config, getApiUrl } from "../../lib/config";

export default function Home() {
  const [dashboardData, setDashboardData] = useState<any>(null);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let ws: WebSocket;
    let pollInterval: NodeJS.Timeout = null;

    const fetchDashboardData = async () => {
      try {
        const response = await fetch(getApiUrl("/api/dashboard/summary"));
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        setDashboardData(data);
        setError(null);
      } catch (err) {
        setError("Failed to load dashboard data");
        setDashboardData({
          costs: { breakdown: [], total_cost: 0 },
          duplicates: [],
          cache_recommendations: [],
          anomalies: [],
        });
      }
    };

    fetchDashboardData();
    pollInterval = setInterval(fetchDashboardData, 3000);

    try {
      ws = new WebSocket(config.wsUrl);

      ws.onopen = () => setIsConnected(true);
      ws.onclose = () => setIsConnected(false);
      ws.onerror = () => setIsConnected(false);

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          fetchDashboardData();
        } catch {}
      };

      setWs(ws);

      return () => {
        if (ws) ws.close();
        clearInterval(pollInterval);
      };
    } catch {}
  }, []);

  if (!dashboardData) {
    return (
      <div className="flex flex-col items-center justify-center h-screen bg-background">
        <span className="text-xl text-muted-foreground animate-pulse">
          Loading API Observatory...
        </span>
      </div>
    );
  }

  return (
    <main className="min-h-screen bg-background p-8">
      <div className="max-w-7xl mx-auto space-y-8">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <h1 className="text-4xl font-extrabold tracking-tight">
              API Observatory
            </h1>
            <Badge variant={isConnected ? "secondary" : "destructive"}>
              {isConnected ? "LIVE" : "OFFLINE"}
            </Badge>
          </div>
          <button className="rounded-md px-3 py-2 bg-primary text-primary-foreground shadow hover:bg-primary/90 transition">
            New Monitor
          </button>
        </div>

        {error && <Alert variant="destructive">{error}</Alert>}

        <Tabs defaultValue="metrics" className="w-full">
          <TabsList className="mb-4">
            <TabsTrigger value="metrics">🏠 Overview</TabsTrigger>
            <TabsTrigger value="cost">💸 Cost</TabsTrigger>
            <TabsTrigger value="optimizations">⚡ Optimization</TabsTrigger>
          </TabsList>
          <TabsContent value="metrics">
            <Card>
              <CardHeader>
                <CardTitle>Real-Time Metrics</CardTitle>
              </CardHeader>
              <CardContent>
                <RealTimeMetrics data={dashboardData} />
              </CardContent>
            </Card>
          </TabsContent>
          <TabsContent value="cost">
            <CostDashboard costs={dashboardData.costs} />
          </TabsContent>
          <TabsContent value="optimizations">
            <OptimizationPanel
              duplicates={dashboardData.duplicates}
              cacheRecommendations={dashboardData.cache_recommendations}
              anomalies={dashboardData.anomalies}
            />
          </TabsContent>
        </Tabs>
        <Card className="mt-6">
          <CardHeader>
            <CardTitle>Status</CardTitle>
          </CardHeader>
          <CardContent>
            <Progress value={isConnected ? 100 : 10} className="w-full" />
            <span className="text-xs text-muted-foreground mt-2 block">
              {isConnected
                ? "You are receiving real-time updates."
                : "Dashboard is offline or disconnected."}
            </span>
          </CardContent>
        </Card>
      </div>
    </main>
  );
}
