export type FieldState = 'idle' | 'checking' | 'ok' | 'warn' | 'error';

export type PasswordStrength = 'empty' | 'too-short' | 'weak' | 'fair' | 'good' | 'strong';

const COMMON_PASSWORDS = new Set([
  'password',
  'password1',
  'password123',
  '12345678',
  '123456789',
  'qwerty123',
  'letmein1',
  'welcome1',
  'puttertalk',
]);

export function assessPassword(password: string): { strength: PasswordStrength; message: string } {
  const n = [...password].length;
  if (n === 0) return { strength: 'empty', message: '' };
  if (n < 8) return { strength: 'too-short', message: 'At least 8 characters' };

  const lower = password.toLowerCase();
  if (COMMON_PASSWORDS.has(lower)) {
    return { strength: 'weak', message: 'Too common — try something harder to guess' };
  }
  if (/^(.)\1{5,}$/.test(password)) {
    return { strength: 'weak', message: 'Repeated characters are easy to guess' };
  }

  const hasLower = /[a-z]/.test(password);
  const hasUpper = /[A-Z]/.test(password);
  const hasDigit = /\d/.test(password);
  const hasSymbol = /[^a-zA-Z0-9]/.test(password);
  const variety = [hasLower, hasUpper, hasDigit, hasSymbol].filter(Boolean).length;

  if (variety >= 3 && n >= 12) {
    return { strength: 'strong', message: 'Strong password' };
  }
  if (variety >= 2 && n >= 8) {
    return { strength: 'good', message: 'Good — mix in a symbol or two for extra strength' };
  }
  if (n >= 8) {
    return { strength: 'fair', message: 'Fair — add numbers or mixed case' };
  }
  return { strength: 'weak', message: 'Weak — make it longer and less predictable' };
}

export function strengthClass(strength: PasswordStrength): string {
  switch (strength) {
    case 'too-short':
    case 'weak':
      return 'field-hint--error';
    case 'fair':
      return 'field-hint--warn';
    case 'good':
    case 'strong':
      return 'field-hint--ok';
    default:
      return '';
  }
}

export function wrapPasswordField(input: HTMLInputElement): void {
  if (input.closest('.password-wrap')) return;

  const wrap = document.createElement('div');
  wrap.className = 'password-wrap';
  input.parentNode?.insertBefore(wrap, input);
  wrap.appendChild(input);

  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'password-reveal';
  btn.setAttribute('aria-label', 'Show password');
  btn.innerHTML = `<span class="password-reveal__show" aria-hidden="true">Show</span><span class="password-reveal__hide" aria-hidden="true" hidden>Hide</span>`;
  wrap.appendChild(btn);

  btn.addEventListener('click', () => {
    const showing = input.type === 'text';
    input.type = showing ? 'password' : 'text';
    const show = btn.querySelector('.password-reveal__show') as HTMLElement;
    const hide = btn.querySelector('.password-reveal__hide') as HTMLElement;
    show.hidden = !showing;
    hide.hidden = showing;
    btn.setAttribute('aria-label', showing ? 'Show password' : 'Hide password');
  });
}

function setFieldState(input: HTMLInputElement, state: FieldState) {
  input.classList.remove('field--ok', 'field--warn', 'field--error', 'field--shake');
  if (state === 'ok') input.classList.add('field--ok');
  if (state === 'warn') input.classList.add('field--warn');
  if (state === 'error') {
    input.classList.add('field--error');
    void input.offsetWidth;
    input.classList.add('field--shake');
  }
}

function debounce<T extends (...args: never[]) => void>(fn: T, ms: number): T {
  let t: ReturnType<typeof setTimeout> | undefined;
  return ((...args: never[]) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...args), ms);
  }) as T;
}

export function initDisplayNameCheck(input: HTMLInputElement, hint: HTMLElement) {
  let lastQuery = '';

  const run = debounce(async () => {
    const name = input.value.trim();
    if (!name) {
      hint.textContent = '';
      hint.hidden = true;
      setFieldState(input, 'idle');
      return;
    }
    if (name.length < 2) {
      hint.textContent = 'At least 2 characters';
      hint.hidden = false;
      hint.className = 'field-hint field-hint--warn';
      setFieldState(input, 'warn');
      return;
    }

    lastQuery = name;
    hint.textContent = 'Checking…';
    hint.hidden = false;
    hint.className = 'field-hint field-hint--checking';
    setFieldState(input, 'checking');

    try {
      const res = await fetch(`/api/auth/check-display-name?name=${encodeURIComponent(name)}`);
      const data = await res.json();
      if (input.value.trim() !== lastQuery) return;

      if (!res.ok) {
        hint.textContent = data.error || 'Could not check name';
        hint.className = 'field-hint field-hint--warn';
        setFieldState(input, 'warn');
        return;
      }

      if (data.available) {
        const slug = data.slug ? ` (@${data.slug})` : '';
        hint.textContent = `Available${slug}`;
        hint.className = 'field-hint field-hint--ok';
        setFieldState(input, 'ok');
      } else if (data.reason === 'reserved') {
        hint.textContent = 'This name is reserved — pick something else';
        hint.className = 'field-hint field-hint--error';
        setFieldState(input, 'error');
      } else {
        hint.textContent = 'Already taken — names are case-insensitive';
        hint.className = 'field-hint field-hint--error';
        setFieldState(input, 'error');
      }
    } catch {
      hint.textContent = '';
      hint.hidden = true;
      setFieldState(input, 'idle');
    }
  }, 350);

  input.addEventListener('input', run);
  input.addEventListener('blur', run);
}

export function initPasswordStrength(input: HTMLInputElement, hint: HTMLElement) {
  const update = () => {
    const { strength, message } = assessPassword(input.value);
    if (!message) {
      hint.textContent = '';
      hint.hidden = true;
      setFieldState(input, 'idle');
      return;
    }
    hint.textContent = message;
    hint.hidden = false;
    hint.className = `field-hint ${strengthClass(strength)}`;
    if (strength === 'too-short' || strength === 'weak') {
      setFieldState(input, 'error');
    } else if (strength === 'fair') {
      setFieldState(input, 'warn');
    } else {
      setFieldState(input, 'ok');
    }
  };
  input.addEventListener('input', update);
  update();
}

export function isPasswordAcceptable(password: string): boolean {
  const { strength } = assessPassword(password);
  return strength !== 'empty' && strength !== 'too-short';
}

