export type ToastVariant = 'info' | 'success' | 'warning' | 'error';

type ConfirmOpts = {
  title?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  dangerous?: boolean;
};

type PromptOpts = {
  title?: string;
  defaultValue?: string;
  placeholder?: string;
  confirmLabel?: string;
  cancelLabel?: string;
};

let confirmResolve: ((value: boolean) => void) | null = null;
let promptResolve: ((value: string | null) => void) | null = null;

function host() {
  return document.getElementById('pt-toast-host');
}

export function showToast(message: string, variant: ToastVariant = 'info', durationMs = 4200) {
  const root = host()?.querySelector('.pt-toast-stack');
  if (!root) return;

  const el = document.createElement('div');
  el.className = `pt-toast pt-toast--${variant}`;
  el.setAttribute('role', 'status');
  el.textContent = message;
  root.append(el);

  const dismiss = () => {
    el.classList.add('pt-toast--out');
    window.setTimeout(() => el.remove(), 220);
  };

  let timer = window.setTimeout(dismiss, variant === 'error' ? 5600 : durationMs);
  el.addEventListener('mouseenter', () => window.clearTimeout(timer));
  el.addEventListener('mouseleave', () => {
    timer = window.setTimeout(dismiss, 1800);
  });
}

export function confirm(message: string, opts: ConfirmOpts = {}): Promise<boolean> {
  const dialog = document.getElementById('pt-confirm');
  const msgEl = document.getElementById('pt-confirm-msg');
  const titleEl = document.getElementById('pt-confirm-title');
  const okBtn = document.getElementById('pt-confirm-ok') as HTMLButtonElement | null;
  const cancelBtn = document.getElementById('pt-confirm-cancel') as HTMLButtonElement | null;
  if (!dialog || !msgEl || !okBtn || !cancelBtn) return Promise.resolve(false);

  if (titleEl) titleEl.textContent = opts.title ?? 'Confirm';
  msgEl.textContent = message;
  okBtn.textContent = opts.confirmLabel ?? 'Confirm';
  cancelBtn.textContent = opts.cancelLabel ?? 'Cancel';
  okBtn.classList.toggle('btn--pink', !opts.dangerous);
  okBtn.classList.toggle('pt-dialog-btn--danger', !!opts.dangerous);

  dialog.hidden = false;
  document.body.classList.add('dialog-open');
  cancelBtn.focus();

  return new Promise((resolve) => {
    confirmResolve = resolve;

    const finish = (value: boolean) => {
      dialog.hidden = true;
      document.body.classList.remove('dialog-open');
      okBtn.onclick = null;
      cancelBtn.onclick = null;
      const res = confirmResolve;
      confirmResolve = null;
      res?.(value);
      resolve(value);
    };

    const onOk = () => finish(true);
    const onCancel = () => finish(false);

    okBtn.onclick = onOk;
    cancelBtn.onclick = onCancel;

    const onKey = (e: KeyboardEvent) => {
      if (dialog.hidden) return;
      if (e.key === 'Escape') onCancel();
      if (e.key === 'Enter') onOk();
    };
    document.addEventListener('keydown', onKey, { once: false });

    const observer = new MutationObserver(() => {
      if (dialog.hidden) {
        document.removeEventListener('keydown', onKey);
        observer.disconnect();
      }
    });
    observer.observe(dialog, { attributes: true, attributeFilter: ['hidden'] });
  });
}

export function promptDialog(message: string, opts: PromptOpts = {}): Promise<string | null> {
  const dialog = document.getElementById('pt-prompt');
  const msgEl = document.getElementById('pt-prompt-msg');
  const titleEl = document.getElementById('pt-prompt-title');
  const input = document.getElementById('pt-prompt-input') as HTMLInputElement | null;
  const okBtn = document.getElementById('pt-prompt-ok') as HTMLButtonElement | null;
  const cancelBtn = document.getElementById('pt-prompt-cancel') as HTMLButtonElement | null;
  if (!dialog || !msgEl || !input || !okBtn || !cancelBtn) return Promise.resolve(null);

  if (titleEl) titleEl.textContent = opts.title ?? message;
  msgEl.textContent = opts.title ? message : '';
  msgEl.hidden = !opts.title;
  input.value = opts.defaultValue ?? '';
  input.placeholder = opts.placeholder ?? '';
  okBtn.textContent = opts.confirmLabel ?? 'OK';
  cancelBtn.textContent = opts.cancelLabel ?? 'Cancel';

  dialog.hidden = false;
  document.body.classList.add('dialog-open');
  input.focus();
  input.select();

  return new Promise((resolve) => {
    promptResolve = resolve;

    const finish = (value: string | null) => {
      dialog.hidden = true;
      document.body.classList.remove('dialog-open');
      okBtn.onclick = null;
      cancelBtn.onclick = null;
      input.onkeydown = null;
      const res = promptResolve;
      promptResolve = null;
      res?.(value);
      resolve(value);
    };

    const onOk = () => finish(input.value);
    const onCancel = () => finish(null);

    okBtn.onclick = onOk;
    cancelBtn.onclick = onCancel;
    input.onkeydown = (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        onOk();
      }
    };

    const onKey = (e: KeyboardEvent) => {
      if (dialog.hidden) return;
      if (e.key === 'Escape') onCancel();
    };
    document.addEventListener('keydown', onKey);

    const observer = new MutationObserver(() => {
      if (dialog.hidden) {
        document.removeEventListener('keydown', onKey);
        observer.disconnect();
      }
    });
    observer.observe(dialog, { attributes: true, attributeFilter: ['hidden'] });
  });
}

export function initToastHost() {
  document.getElementById('pt-confirm-backdrop')?.addEventListener('click', () => {
    document.getElementById('pt-confirm-cancel')?.click();
  });

  document.getElementById('pt-prompt-backdrop')?.addEventListener('click', () => {
    document.getElementById('pt-prompt-cancel')?.click();
  });
}