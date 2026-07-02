const OUT_SIZE = 512;

interface CropState {
  img: HTMLImageElement;
  vpSize: number;
  zoom: number;
  panX: number;
  panY: number;
  fit: number;
}

function fitScale(img: HTMLImageElement, vpSize: number): number {
  return Math.max(vpSize / img.naturalWidth, vpSize / img.naturalHeight);
}

function clampPan(state: CropState): void {
  const scale = state.fit * state.zoom;
  const drawW = state.img.naturalWidth * scale;
  const drawH = state.img.naturalHeight * scale;
  const maxX = Math.max(0, (drawW - state.vpSize) / 2);
  const maxY = Math.max(0, (drawH - state.vpSize) / 2);
  state.panX = Math.min(maxX, Math.max(-maxX, state.panX));
  state.panY = Math.min(maxY, Math.max(-maxY, state.panY));
}

function applyTransform(img: HTMLImageElement, state: CropState): void {
  const scale = state.fit * state.zoom;
  img.style.width = `${state.img.naturalWidth * scale}px`;
  img.style.height = `${state.img.naturalHeight * scale}px`;
  img.style.transform = `translate(calc(-50% + ${state.panX}px), calc(-50% + ${state.panY}px))`;
}

async function exportCrop(state: CropState): Promise<Blob> {
  const scale = state.fit * state.zoom;
  const drawX = state.vpSize / 2 - (state.img.naturalWidth * scale) / 2 + state.panX;
  const drawY = state.vpSize / 2 - (state.img.naturalHeight * scale) / 2 + state.panY;
  const sx = -drawX / scale;
  const sy = -drawY / scale;
  const sw = state.vpSize / scale;
  const sh = state.vpSize / scale;

  const canvas = document.createElement('canvas');
  canvas.width = OUT_SIZE;
  canvas.height = OUT_SIZE;
  const ctx = canvas.getContext('2d');
  if (!ctx) throw new Error('Canvas unavailable');

  ctx.drawImage(state.img, sx, sy, sw, sh, 0, 0, OUT_SIZE, OUT_SIZE);

  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error('Could not encode image'))),
      'image/jpeg',
      0.92,
    );
  });
}

function loadImage(file: File): Promise<{ img: HTMLImageElement; url: string }> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => resolve({ img, url });
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Could not load image'));
    };
    img.src = url;
  });
}

/** Lightbox crop UI — drag to position, slider to zoom. Returns JPEG blob or null if cancelled. */
export async function openAvatarCrop(file: File): Promise<Blob | null> {
  const { img, url: objectUrl } = await loadImage(file);

  const root = document.createElement('div');
  root.className = 'avatar-crop';
  root.innerHTML = `
    <div class="avatar-crop-backdrop"></div>
    <div class="avatar-crop-panel" role="dialog" aria-modal="true" aria-labelledby="avatar-crop-title">
      <h3 id="avatar-crop-title">Adjust your photo</h3>
      <p class="avatar-crop-hint">Drag to reposition · use the slider to zoom</p>
      <div class="avatar-crop-viewport" tabindex="0">
        <img class="avatar-crop-img" alt="" draggable="false" />
      </div>
      <label class="avatar-crop-zoom">
        <span>Zoom</span>
        <input type="range" min="1" max="3" step="0.01" value="1" />
      </label>
      <div class="avatar-crop-actions">
        <button type="button" class="btn btn--sm btn--ghost" data-crop-cancel>Cancel</button>
        <button type="button" class="btn btn--sm btn--pink" data-crop-save>Save photo</button>
      </div>
    </div>
  `;
  document.body.append(root);
  document.body.style.overflow = 'hidden';

  const viewport = root.querySelector('.avatar-crop-viewport') as HTMLDivElement;
  const cropImg = root.querySelector('.avatar-crop-img') as HTMLImageElement;
  const slider = root.querySelector('input[type="range"]') as HTMLInputElement;
  const cancelBtn = root.querySelector('[data-crop-cancel]') as HTMLButtonElement;
  const saveBtn = root.querySelector('[data-crop-save]') as HTMLButtonElement;
  const backdrop = root.querySelector('.avatar-crop-backdrop') as HTMLDivElement;

  cropImg.src = objectUrl;

  const state: CropState = {
    img,
    vpSize: 0,
    zoom: 1,
    panX: 0,
    panY: 0,
    fit: 1,
  };

  const measure = () => {
    state.vpSize = viewport.clientWidth;
    state.fit = fitScale(img, state.vpSize);
    clampPan(state);
    applyTransform(cropImg, state);
  };

  requestAnimationFrame(measure);

  let dragging = false;
  let startX = 0;
  let startY = 0;
  let originPanX = 0;
  let originPanY = 0;

  const onPointerDown = (e: PointerEvent) => {
    dragging = true;
    startX = e.clientX;
    startY = e.clientY;
    originPanX = state.panX;
    originPanY = state.panY;
    viewport.setPointerCapture(e.pointerId);
  };

  const onPointerMove = (e: PointerEvent) => {
    if (!dragging) return;
    state.panX = originPanX + (e.clientX - startX);
    state.panY = originPanY + (e.clientY - startY);
    clampPan(state);
    applyTransform(cropImg, state);
  };

  const onPointerUp = (e: PointerEvent) => {
    dragging = false;
    try {
      viewport.releasePointerCapture(e.pointerId);
    } catch {
      /* ignore */
    }
  };

  viewport.addEventListener('pointerdown', onPointerDown);
  viewport.addEventListener('pointermove', onPointerMove);
  viewport.addEventListener('pointerup', onPointerUp);
  viewport.addEventListener('pointercancel', onPointerUp);

  slider.addEventListener('input', () => {
    state.zoom = Number(slider.value);
    clampPan(state);
    applyTransform(cropImg, state);
  });

  const cleanup = () => {
    viewport.removeEventListener('pointerdown', onPointerDown);
    viewport.removeEventListener('pointermove', onPointerMove);
    viewport.removeEventListener('pointerup', onPointerUp);
    viewport.removeEventListener('pointercancel', onPointerUp);
    URL.revokeObjectURL(objectUrl);
    root.remove();
    document.body.style.overflow = '';
  };

  return new Promise((resolve) => {
    const finish = (blob: Blob | null) => {
      cleanup();
      resolve(blob);
    };

    cancelBtn.addEventListener('click', () => finish(null));
    backdrop.addEventListener('click', () => finish(null));

    root.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') finish(null);
    });

    saveBtn.addEventListener('click', async () => {
      saveBtn.disabled = true;
      try {
        const blob = await exportCrop(state);
        finish(blob);
      } catch {
        saveBtn.disabled = false;
      }
    });
  });
}