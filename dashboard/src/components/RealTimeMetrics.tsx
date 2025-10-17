"use client";

import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";

export default function RealTimeMetrics({ data }: { data: any }) {
  const totalRequests =
    data?.costs?.breakdown?.reduce(
      (sum: number, row: any) => sum + row.request_count,
      0,
    ) || 0;
  const totalErrors =
    data?.costs?.breakdown?.reduce(
      (sum: number, row: any) => sum + row.error_count,
      0,
    ) || 0;
  const topProvider =
    data?.costs?.breakdown && data.costs.breakdown[0]
      ? data.costs.breakdown[0].label
      : "–";

  return (
    <Card>
      <CardHeader>
        <CardTitle>📊 Real-Time Metrics</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-6 mb-4">
          <div>
            <div className="font-semibold text-muted-foreground text-xs">
              Total Requests
            </div>
            <div className="text-2xl font-bold">{totalRequests}</div>
          </div>
          <div>
            <div className="font-semibold text-muted-foreground text-xs">
              Errors
            </div>
            <div className="text-2xl font-bold text-destructive">
              {totalErrors}
            </div>
          </div>
        </div>
        <div className="mb-2">
          <div className="font-semibold text-xs">Top API Provider</div>
          <Badge variant="secondary">{topProvider}</Badge>
        </div>
        <div className="mt-2">
          <Progress
            value={
              totalErrors > 0
                ? (100 * totalErrors) / Math.max(totalRequests, 1)
                : 100
            }
          />
        </div>
      </CardContent>
    </Card>
  );
}
