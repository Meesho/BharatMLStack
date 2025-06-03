export async function loadEnv() {
  try {
    await new Promise((resolve, reject) => {
      const script = document.createElement('script');
      script.src = '/env.js';
      script.onload = resolve;
      script.onerror = reject;
      document.head.appendChild(script);
    });
  } catch (error) {
    console.error('Failed to load env.js', error);
  }
}

  