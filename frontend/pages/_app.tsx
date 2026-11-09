import type { AppProps } from 'next/app';
import Head from 'next/head';
import '../styles/global.css';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <>
      <Head>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath d='M2.5 5h19L12 20.5z' fill='%23000'/%3E%3C/svg%3E" />
      </Head>
      <Component {...pageProps} />
    </>
  );
}
