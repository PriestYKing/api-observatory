"use client";

import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";

export default function CostDashboard({ costs }: { costs: any }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>
          💸 Cost Analysis
          <Badge className="ml-2" variant="outline">
            {costs.total_cost ? `$${costs.total_cost.toFixed(2)}` : "$0.00"}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Provider</TableHead>
              <TableHead>Requests</TableHead>
              <TableHead>Cost</TableHead>
              <TableHead>Avg Latency</TableHead>
              <TableHead>Errors</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {costs.breakdown && costs.breakdown.length > 0 ? (
              costs.breakdown.map((row: any, i: number) => (
                <TableRow key={i}>
                  <TableCell>
                    <Badge variant="secondary">{row.label}</Badge>
                  </TableCell>
                  <TableCell>{row.request_count}</TableCell>
                  <TableCell>${row.cost.toFixed(4)}</TableCell>
                  <TableCell>{row.avg_latency.toFixed(1)}ms</TableCell>
                  <TableCell>
                    <Badge
                      variant={row.error_count > 0 ? "destructive" : "outline"}
                    >
                      {row.error_count}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={5}
                  className="text-muted-foreground text-center"
                >
                  No data available
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
