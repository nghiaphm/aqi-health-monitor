"use client";

import { useEffect, useState, useCallback, createContext, useContext } from "react";
import { UserState } from "@/types/user";
import { getMe } from "@/lib/api/user";
import { getAuthToken, setAuthToken } from "@/lib/api/client";

interface AuthContextType {
  user: UserState | null;
  token: string | null;
  isLoading: boolean;
  error: Error | null;
  needsOnboarding: boolean;
  loginWithToken: (newToken: string) => Promise<void>;
  logout: () => void;
  refreshUser: () => Promise<UserState | null>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<UserState | null>(null);
  const [token, setTokenState] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchCurrentUser = useCallback(async (): Promise<UserState | null> => {
    try {
      const currentToken = getAuthToken();
      if (!currentToken) {
        setUser(null);
        setIsLoading(false);
        return null;
      }
      setTokenState(currentToken);
      const userData = await getMe();
      setUser(userData);
      setError(null);
      return userData;
    } catch (err: unknown) {
      setUser(null);
      setError(err instanceof Error ? err : new Error(String(err)));
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function init() {
      try {
        const currentToken = getAuthToken();
        if (!currentToken) {
          if (!cancelled) {
            setUser(null);
            setIsLoading(false);
          }
          return;
        }
        if (!cancelled) setTokenState(currentToken);
        const userData = await getMe();
        if (!cancelled) {
          setUser(userData);
          setError(null);
          setIsLoading(false);
        }
      } catch (err: unknown) {
        if (!cancelled) {
          setUser(null);
          setError(err instanceof Error ? err : new Error(String(err)));
          setIsLoading(false);
        }
      }
    }

    init();

    return () => {
      cancelled = true;
    };
  }, []);

  const loginWithToken = useCallback(
    async (newToken: string) => {
      setAuthToken(newToken);
      setTokenState(newToken);
      setIsLoading(true);
      await fetchCurrentUser();
    },
    [fetchCurrentUser]
  );

  const logout = useCallback(() => {
    setAuthToken(null);
    setTokenState(null);
    setUser(null);
  }, []);

  const needsOnboarding = Boolean(user && !user.health_profile);

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        isLoading,
        error,
        needsOnboarding,
        loginWithToken,
        logout,
        refreshUser: fetchCurrentUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
