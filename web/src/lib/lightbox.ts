export function initLightbox(scope: ParentNode = document) {
  const host = document.getElementById('pt-lightbox');
  if (!host) return;

  const imgEl = host.querySelector<HTMLImageElement>('.pt-lightbox-img');
  const counterEl = host.querySelector<HTMLElement>('.pt-lightbox-counter');
  const prevBtn = host.querySelector<HTMLButtonElement>('.pt-lightbox-prev');
  const nextBtn = host.querySelector<HTMLButtonElement>('.pt-lightbox-next');
  const closeBtn = host.querySelector<HTMLButtonElement>('.pt-lightbox-close');
  const backdrop = host.querySelector<HTMLElement>('.pt-lightbox-backdrop');

  if (!imgEl || !counterEl || !prevBtn || !nextBtn || !closeBtn || !backdrop) return;

  let activeImages: HTMLImageElement[] = [];
  let index = 0;

  scope.querySelectorAll('.post-body').forEach((postBody) => {
    const images = Array.from(postBody.querySelectorAll<HTMLImageElement>('img'));
    if (images.length === 0) return;

    images.forEach((img, i) => {
      img.classList.add('post-inline-img');
      img.tabIndex = 0;
      img.addEventListener('click', (e) => {
        e.preventDefault();
        open(images, i);
      });
      img.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          open(images, i);
        }
      });
    });
  });

  function imageSrc(img: HTMLImageElement): string {
    return img.getAttribute('data-full-src') || img.getAttribute('src') || '';
  }

  function isMulti(): boolean {
    return activeImages.length > 1;
  }

  function show(i: number) {
    if (activeImages.length === 0) return;
    index = (i + activeImages.length) % activeImages.length;
    const img = activeImages[index];
    const fullSrc = imageSrc(img);
    imgEl.removeAttribute('style');
    imgEl.removeAttribute('width');
    imgEl.removeAttribute('height');
    imgEl.classList.add('is-loading');
    imgEl.onload = () => imgEl.classList.remove('is-loading');
    imgEl.onerror = () => imgEl.classList.remove('is-loading');
    imgEl.src = fullSrc;
    imgEl.alt = img.getAttribute('alt') || '';
    const multi = isMulti();
    counterEl.textContent = multi ? `${index + 1} / ${activeImages.length}` : '';
    counterEl.hidden = !multi;
    prevBtn.hidden = !multi;
    nextBtn.hidden = !multi;
  }

  function open(images: HTMLImageElement[], i: number) {
    activeImages = images;
    show(i);
    host.hidden = false;
    document.body.classList.add('lightbox-open');
    closeBtn.focus();
  }

  function close() {
    host.hidden = true;
    imgEl.removeAttribute('src');
    imgEl.removeAttribute('style');
    imgEl.classList.remove('is-loading');
    activeImages = [];
    document.body.classList.remove('lightbox-open');
  }

  function step(delta: number) {
    if (!isMulti()) return;
    show(index + delta);
  }

  prevBtn.addEventListener('click', () => step(-1));
  nextBtn.addEventListener('click', () => step(1));
  closeBtn.addEventListener('click', close);
  backdrop.addEventListener('click', close);

  document.addEventListener('keydown', (e) => {
    if (host.hidden) return;
    if (e.key === 'Escape') close();
    if (e.key === 'ArrowLeft') step(-1);
    if (e.key === 'ArrowRight') step(1);
  });
}