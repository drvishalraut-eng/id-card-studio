// Resizes an image file client-side before any photo upload, per spec:
// "resize photos to a maximum of 800 px on the long side, as JPEG at
// quality 0.9, before uploading." Also reports the original (pre-resize)
// dimensions, so callers can warn when the source photo's short side is
// under 600 px (Step 3).
export function resizeImageToJpeg(file, maxSide = 800, quality = 0.9) {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      const originalWidth = img.naturalWidth;
      const originalHeight = img.naturalHeight;
      let width = originalWidth;
      let height = originalHeight;
      const longSide = Math.max(width, height);
      if (longSide > maxSide) {
        const scale = maxSide / longSide;
        width = Math.round(width * scale);
        height = Math.round(height * scale);
      }

      const canvas = document.createElement('canvas');
      canvas.width = width;
      canvas.height = height;
      const ctx = canvas.getContext('2d');
      ctx.drawImage(img, 0, 0, width, height);
      URL.revokeObjectURL(url);

      canvas.toBlob(
        (blob) => {
          if (blob) resolve({ blob, originalWidth, originalHeight });
          else reject(new Error('Could not process the photo.'));
        },
        'image/jpeg',
        quality,
      );
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Could not read the photo file.'));
    };
    img.src = url;
  });
}
