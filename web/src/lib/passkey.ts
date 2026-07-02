import {
  browserSupportsWebAuthn,
  startAuthentication,
  startRegistration,
} from '@simplewebauthn/browser';
import type {
  AuthenticationResponseJSON,
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
  RegistrationResponseJSON,
} from '@simplewebauthn/browser';

type BeginResponse = {
  session_id: string;
  options: {
    publicKey?: PublicKeyCredentialCreationOptionsJSON;
    response?: PublicKeyCredentialRequestOptionsJSON;
  };
};

/** Run before ceremony — returns a user-facing warning or null if OK. */
export async function checkPasskeySupport(): Promise<string | null> {
  if (!browserSupportsWebAuthn()) {
    return 'Passkeys are not supported in this browser.';
  }
  try {
    const platform = await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
    if (!platform) {
      return 'No Touch ID / Face ID authenticator found. Try Safari, or use password sign-in.';
    }
  } catch {
    /* optional API — ignore */
  }
  return null;
}

function passkeyErrorMessage(err: unknown): string {
  if (err instanceof Error) {
    const msg = err.message;
    if (
      msg.includes('not allowed') ||
      msg.includes('timed out') ||
      msg.includes('NotAllowedError')
    ) {
      return (
        'Passkey was blocked or timed out. In Chrome: sign into your Google account, ' +
        'or use Safari (iCloud Keychain). For local dev: Chrome DevTools → Application → ' +
        'WebAuthn → enable a virtual authenticator. Password sign-in always works.'
      );
    }
    if (msg.includes('Google Password Manager') || msg.includes('password manager')) {
      return (
        'Chrome could not reach Google Password Manager. Sign into Chrome/Google, ' +
        'try Safari, or use password sign-in.'
      );
    }
    return msg;
  }
  return 'Passkey failed';
}

function creationOptionsJSON(raw: BeginResponse['options']): PublicKeyCredentialCreationOptionsJSON {
  const pk = raw.publicKey;
  if (!pk) throw new Error('Missing passkey registration options from server');
  return pk;
}

function requestOptionsJSON(raw: BeginResponse['options']): PublicKeyCredentialRequestOptionsJSON {
  const pk = raw.publicKey;
  if (!pk) throw new Error('Missing passkey login options from server');
  return pk;
}

export async function passkeyCeremony(
  beginUrl: string,
  finishUrl: string,
  beginBody?: Record<string, unknown>,
): Promise<{ ok: boolean; error?: string }> {
  const preflight = await checkPasskeySupport();
  if (preflight) {
    return { ok: false, error: preflight };
  }

  const beginRes = await fetch(beginUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      ...(beginBody ?? {}),
      origin: window.location.origin,
    }),
  });
  const beginData = (await beginRes.json()) as BeginResponse & { error?: string };
  if (!beginRes.ok) {
    return { ok: false, error: beginData.error || 'Could not start passkey flow' };
  }

  const useCreate = beginUrl.includes('signup') || beginUrl.includes('register');
  let credential: RegistrationResponseJSON | AuthenticationResponseJSON;

  try {
    if (useCreate) {
      credential = await startRegistration({
        optionsJSON: creationOptionsJSON(beginData.options),
      });
    } else {
      credential = await startAuthentication({
        optionsJSON: requestOptionsJSON(beginData.options),
      });
    }
  } catch (err) {
    return { ok: false, error: passkeyErrorMessage(err) };
  }

  const finishRes = await fetch(finishUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      session_id: beginData.session_id,
      credential,
    }),
  });
  const finishData = await finishRes.json();
  if (!finishRes.ok) {
    if (finishRes.status === 403 && finishData.ban_reason) {
      try {
        sessionStorage.setItem('pt-ban', JSON.stringify(finishData));
      } catch {
        /* ignore */
      }
      window.location.href = '/banned';
      return { ok: false };
    }
    return { ok: false, error: finishData.error || 'Passkey verification failed' };
  }
  return { ok: true };
}