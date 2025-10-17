"use client";

import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";

export default function OptimizationPanel({
  duplicates,
  cacheRecommendations,
  anomalies,
}: {
  duplicates: any[];
  cacheRecommendations: any[];
  anomalies: any[];
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>⚡ Optimization Opportunities</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {/* Cache Recommendations */}
          <div>
            <div className="font-semibold">Potential Savings 💰</div>
            {cacheRecommendations.length > 0 ? (
              cacheRecommendations.map((rec, i) => (
                <Alert key={i} variant="default" className="mb-2">
                  <AlertTitle>
                    {rec.endpoint} —
                    <Badge className="ml-1" variant="secondary">
                      Save ≈ ${rec.potential_savings.toFixed(2)}
                    </Badge>
                  </AlertTitle>
                  <AlertDescription>{rec.recommendation}</AlertDescription>
                </Alert>
              ))
            ) : (
              <div className="text-muted-foreground text-sm">
                No cache opportunities detected
              </div>
            )}
          </div>
          {/* Duplicates */}
          <div>
            <div className="font-semibold">Duplicate Patterns 🌀</div>
            {duplicates.length > 0 ? (
              duplicates.map((dup, i) => (
                <Alert key={i} variant="destructive" className="mb-2">
                  <AlertTitle>
                    {dup.endpoint}
                    <Badge className="ml-2" variant="outline">
                      {dup.count}x
                    </Badge>
                  </AlertTitle>
                  <AlertDescription>
                    {dup.first_seen
                      ? `First Seen: ${new Date(dup.first_seen).toLocaleString()}`
                      : null}
                  </AlertDescription>
                </Alert>
              ))
            ) : (
              <div className="text-muted-foreground text-sm">
                No duplicates detected
              </div>
            )}
          </div>
          {/* Anomalies */}
          <div>
            <div className="font-semibold">Recent Anomalies 🚨</div>
            {anomalies.length > 0 ? (
              anomalies.map((anomaly, i) => (
                <Alert key={i} variant="destructive" className="mb-2">
                  <AlertTitle>{anomaly.type}</AlertTitle>
                  <AlertDescription>
                    <span className="font-bold">{anomaly.severity}</span>:{" "}
                    {anomaly.description}
                    <br />
                    Detected at:{" "}
                    {new Date(anomaly.detected_at).toLocaleString()}
                  </AlertDescription>
                </Alert>
              ))
            ) : (
              <div className="text-muted-foreground text-sm">
                No anomalies detected
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
