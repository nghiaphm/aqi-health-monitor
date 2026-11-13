"use client";

import { useEffect, useState } from "react";
import { getCurrentAQI } from "@/lib/api/aqi";
import { CurrentAQIResponse } from "@/types/aqi";

export function useAQI(locationId?: string) {
  const [data, setData] = useState<CurrentAQIResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const res = await getCurrentAQI(locationId);
        if (!cancelled) {
          setData(res);
          setLoading(false);
        }
      } catch (err: unknown) {
        if (!cancelled) {
          setError(err instanceof Error ? err : new Error(String(err)));
          setLoading(false);
        }
      }
    }

    load();

    return () => {
      cancelled = true;
    };
  }, [locationId]);

  return { data, loading, error };
}
