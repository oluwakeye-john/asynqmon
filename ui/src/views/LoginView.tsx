import React, { FormEvent, useState } from "react";
import Button from "@material-ui/core/Button";
import CircularProgress from "@material-ui/core/CircularProgress";
import IconButton from "@material-ui/core/IconButton";
import InputAdornment from "@material-ui/core/InputAdornment";
import Paper from "@material-ui/core/Paper";
import TextField from "@material-ui/core/TextField";
import Typography from "@material-ui/core/Typography";
import { makeStyles, Theme, useTheme as useMuiTheme } from "@material-ui/core/styles";
import {
  ErrorOutline as ErrorOutlineIcon,
  LockOutlined as LockOutlinedIcon,
  Visibility as VisibilityIcon,
  VisibilityOff as VisibilityOffIcon,
} from "@material-ui/icons";
import { isDarkTheme } from "../theme";
import Logo from "../images/logo-color.svg?react";
import LogoDarkTheme from "../images/logo-white.svg?react";

const useStyles = makeStyles((theme: Theme) => {
  const dark = isDarkTheme(theme);
  return {
    root: {
      position: "relative",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      width: "100%",
      minHeight: "100vh",
      padding: theme.spacing(3),
      overflow: "hidden",
      background: dark
        ? "radial-gradient(circle at 15% 15%, rgba(67, 121, 255, 0.20), transparent 34%), radial-gradient(circle at 85% 82%, rgba(151, 251, 209, 0.12), transparent 30%), #111318"
        : "radial-gradient(circle at 15% 15%, rgba(67, 121, 255, 0.16), transparent 34%), radial-gradient(circle at 85% 82%, rgba(151, 251, 209, 0.30), transparent 30%), #f5f7f9",
    },
    grid: {
      position: "absolute",
      inset: 0,
      opacity: dark ? 0.1 : 0.2,
      backgroundImage:
        "linear-gradient(rgba(67, 121, 255, 0.16) 1px, transparent 1px), linear-gradient(90deg, rgba(67, 121, 255, 0.16) 1px, transparent 1px)",
      backgroundSize: "48px 48px",
      maskImage:
        "linear-gradient(to bottom, rgba(0, 0, 0, 0.75), transparent 78%)",
    },
    card: {
      position: "relative",
      zIndex: 1,
      width: "100%",
      maxWidth: 430,
      padding: theme.spacing(5),
      borderRadius: 24,
      border: `1px solid ${
        dark ? "rgba(255, 255, 255, 0.10)" : "rgba(67, 121, 255, 0.10)"
      }`,
      background: dark
        ? "rgba(35, 38, 47, 0.92)"
        : "rgba(255, 255, 255, 0.94)",
      boxShadow: dark
        ? "0 28px 80px rgba(0, 0, 0, 0.42)"
        : "0 28px 80px rgba(29, 50, 95, 0.16)",
      backdropFilter: "blur(18px)",
      [theme.breakpoints.down("xs")]: {
        padding: theme.spacing(4, 3),
        borderRadius: 20,
      },
    },
    brand: {
      display: "flex",
      justifyContent: "center",
      marginBottom: theme.spacing(3.5),
    },
    lockBadge: {
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      width: 48,
      height: 48,
      margin: `0 auto ${theme.spacing(2)}px`,
      borderRadius: 15,
      color: theme.palette.primary.main,
      background: dark
        ? "rgba(67, 121, 255, 0.18)"
        : "rgba(67, 121, 255, 0.10)",
    },
    title: {
      fontWeight: 700,
      letterSpacing: "-0.02em",
      textAlign: "center",
    },
    subtitle: {
      maxWidth: 320,
      margin: `${theme.spacing(1)}px auto ${theme.spacing(3.5)}px`,
      color: theme.palette.text.secondary,
      lineHeight: 1.55,
      textAlign: "center",
    },
    form: {
      display: "flex",
      flexDirection: "column",
      gap: theme.spacing(2.25),
    },
    field: {
      "& .MuiOutlinedInput-root": {
        borderRadius: 12,
      },
    },
    error: {
      display: "flex",
      alignItems: "flex-start",
      gap: theme.spacing(1),
      padding: theme.spacing(1.5),
      borderRadius: 12,
      color: dark ? theme.palette.error.light : theme.palette.error.dark,
      background: dark
        ? "rgba(244, 67, 54, 0.12)"
        : "rgba(244, 67, 54, 0.08)",
    },
    errorIcon: {
      flexShrink: 0,
      marginTop: 1,
    },
    submit: {
      minHeight: 50,
      borderRadius: 12,
      fontWeight: 700,
      textTransform: "none",
      boxShadow: "0 10px 24px rgba(67, 121, 255, 0.26)",
    },
    submitProgress: {
      color: "inherit",
    },
  };
});

interface LoginViewProps {
  error?: string;
  onSubmit: (username: string, password: string) => Promise<void>;
}

export default function LoginView({ error, onSubmit }: LoginViewProps) {
  const classes = useStyles();
  const theme = useMuiTheme<Theme>();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!username || !password || submitting) return;
    setSubmitting(true);
    try {
      await onSubmit(username, password);
    } catch {
      // AuthGate owns the user-facing error message.
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className={classes.root}>
      <div className={classes.grid} aria-hidden="true" />
      <Paper className={classes.card} elevation={0}>
        <div className={classes.brand}>
          {isDarkTheme(theme) ? (
            <LogoDarkTheme width={190} height={46} />
          ) : (
            <Logo width={190} height={46} />
          )}
        </div>
        <div className={classes.lockBadge}>
          <LockOutlinedIcon />
        </div>
        <Typography variant="h4" component="h1" className={classes.title}>
          Welcome back
        </Typography>
        <Typography variant="body1" className={classes.subtitle}>
          Sign in to securely monitor queues, workers, and scheduled tasks.
        </Typography>

        <form className={classes.form} onSubmit={handleSubmit} noValidate>
          {error && (
            <div className={classes.error} role="alert">
              <ErrorOutlineIcon className={classes.errorIcon} fontSize="small" />
              <Typography variant="body2">{error}</Typography>
            </div>
          )}
          <TextField
            id="auth-username"
            className={classes.field}
            label="Username"
            name="username"
            variant="outlined"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            inputProps={{ "aria-label": "Username" }}
            autoComplete="username"
            autoFocus
            fullWidth
            required
            disabled={submitting}
          />
          <TextField
            id="auth-password"
            className={classes.field}
            label="Password"
            name="password"
            type={showPassword ? "text" : "password"}
            variant="outlined"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            inputProps={{ "aria-label": "Password" }}
            autoComplete="current-password"
            fullWidth
            required
            disabled={submitting}
            InputProps={{
              endAdornment: (
                <InputAdornment position="end">
                  <IconButton
                    aria-label={showPassword ? "Hide password" : "Show password"}
                    edge="end"
                    onClick={() => setShowPassword((visible) => !visible)}
                    disabled={submitting}
                  >
                    {showPassword ? <VisibilityOffIcon /> : <VisibilityIcon />}
                  </IconButton>
                </InputAdornment>
              ),
            }}
          />
          <Button
            className={classes.submit}
            type="submit"
            variant="contained"
            color="primary"
            size="large"
            disabled={!username || !password || submitting}
            fullWidth
          >
            {submitting ? (
              <CircularProgress className={classes.submitProgress} size={22} />
            ) : (
              "Sign in"
            )}
          </Button>
        </form>

      </Paper>
    </main>
  );
}

export function SessionLoadingView() {
  const classes = useStyles();
  const theme = useMuiTheme<Theme>();
  return (
    <main className={classes.root} aria-label="Checking authentication">
      <div className={classes.grid} aria-hidden="true" />
      <Paper className={classes.card} elevation={0}>
        <div className={classes.brand}>
          {isDarkTheme(theme) ? (
            <LogoDarkTheme width={190} height={46} />
          ) : (
            <Logo width={190} height={46} />
          )}
        </div>
        <div style={{ display: "flex", justifyContent: "center" }}>
          <CircularProgress size={30} />
        </div>
      </Paper>
    </main>
  );
}
