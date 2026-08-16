import React, { ReactNode, useEffect, useState } from "react";
import axios from "axios";
import {
  getAuthSession,
  login,
  logout,
  setCSRFToken,
  setUnauthorizedHandler,
} from "../api";
import LoginView, { SessionLoadingView } from "../views/LoginView";

export interface AuthenticatedSession {
  authEnabled: boolean;
  username: string;
  signingOut: boolean;
  signOut: () => Promise<void>;
}

interface AuthGateProps {
  children: (session: AuthenticatedSession) => ReactNode;
}

type AuthStatus = "loading" | "authenticated" | "unauthenticated";

function errorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const message = error.response?.data?.error;
    if (typeof message === "string") return message;
  }
  return "Unable to reach Asynqmon. Check your connection and try again.";
}

export default function AuthGate({ children }: AuthGateProps) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [authEnabled, setAuthEnabled] = useState(false);
  const [username, setUsername] = useState("");
  const [error, setError] = useState<string>();
  const [signingOut, setSigningOut] = useState(false);

  const markUnauthenticated = () => {
    setCSRFToken();
    setUsername("");
    setStatus("unauthenticated");
  };

  useEffect(() => {
    let active = true;
    setUnauthorizedHandler(() => {
      if (active) markUnauthenticated();
    });

    getAuthSession()
      .then((session) => {
        if (!active) return;
        setAuthEnabled(session.enabled);
        if (session.authenticated) {
          setCSRFToken(session.csrfToken);
          setUsername(session.username || "");
          setStatus("authenticated");
        } else {
          markUnauthenticated();
        }
      })
      .catch((requestError) => {
        if (!active) return;
        setAuthEnabled(true);
        setError(errorMessage(requestError));
        markUnauthenticated();
      });

    return () => {
      active = false;
      setUnauthorizedHandler();
    };
  }, []);

  const signIn = async (enteredUsername: string, password: string) => {
    setError(undefined);
    try {
      const session = await login(enteredUsername, password);
      setAuthEnabled(session.enabled);
      setCSRFToken(session.csrfToken);
      setUsername(session.username || enteredUsername);
      setStatus("authenticated");
    } catch (requestError) {
      setError(errorMessage(requestError));
      throw requestError;
    }
  };

  const signOut = async () => {
    setSigningOut(true);
    try {
      await logout();
    } finally {
      setSigningOut(false);
      markUnauthenticated();
    }
  };

  if (status === "loading") return <SessionLoadingView />;
  if (status === "unauthenticated") {
    return <LoginView error={error} onSubmit={signIn} />;
  }

  return (
    <>
      {children({ authEnabled, username, signingOut, signOut })}
    </>
  );
}
