import Head from 'next/head';
import { useState, useEffect } from 'react';

export default function Home() {
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 60);
    };
    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  return (
    <>
      <Head>
        <title>Controller - AI-Powered Discord Management</title>
        <meta name="description" content="AI-powered Discord server management. Describe what you want, and the AI will do it." />
      </Head>

      <header className="nav">
        <div className="wrap nav-in">
          <a href="/" className="brand">
            <svg className="logo" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M2.5 5h19L12 20.5z" />
            </svg>
            Controller
          </a>
          <nav className="nav-right">
            <nav>
              <a href="/how-it-works">How it works</a>
              <a href="/faq">FAQ</a>
              <a href="/support">Support</a>
            </nav>
            <a href="/dashboard" className="btn btn-primary">
              Open Console
            </a>
          </nav>
        </div>
      </header>

      <main>
        <section className="hero">
          <h1>
            <em>AI-Powered</em> Discord Management
          </h1>
          <p className="lead">
            Describe what you want in plain English, and Controller will do it. 
            Create channels, manage roles, moderate members, and more.
          </p>
          <div className="cta">
            <a href="/dashboard" className="btn btn-primary">
              Try it now
            </a>
            <a href="/how-it-works" className="btn btn-secondary">
              Learn more
            </a>
          </div>
          <div className="demo">
            <div className="bubble">
              Create a welcome channel with rules and info
            </div>
            <div className="row">
              <span className="badge s-executed">
                <i></i>
                Completed
              </span>
              <span className="mono">#welcome</span>
            </div>
            <div className="steps-mini">
              <div><b>1</b> Create category: Welcome</div>
              <div><b>2</b> Create channel: #rules</div>
              <div><b>3</b> Create channel: #info</div>
            </div>
          </div>
        </section>

        <section className="block">
          <div className="wrap">
            <div className="head">
              <span className="eyebrow">How it works</span>
              <h2>Simple. Powerful. Safe.</h2>
              <p>No complex commands. No bot syntax. Just describe what you want.</p>
            </div>
            <div className="steps">
              <article className="step">
                <h3>Describe Your Goal</h3>
                <p>Type what you want in plain English. &quot;Create a music category with voice channels&quot; or &quot;Ban the spammer.&quot;</p>
              </article>
              <article className="step">
                <h3>AI Plans It</h3>
                <p>The AI audits your request for safety, then plans the exact steps needed to achieve your goal.</p>
              </article>
              <article className="step">
                <h3>Bot Executes</h3>
                <p>Your Discord bot carries out each action. Watch the progress in real-time in your console.</p>
              </article>
              <article className="step">
                <h3>Done</h3>
                <p>Each step is logged. You can see exactly what was done and when.</p>
              </article>
            </div>
          </div>
        </section>

        <section className="band">
          <div className="wrap">
            <h2>Ready to Automate Your Server?</h2>
            <p className="lead">
              Join thousands of server owners who trust Controller to manage their Discord communities.
            </p>
            <div className="cta">
              <a href="/dashboard" className="btn btn-primary">
                Open Console
              </a>
              <a href="/faq" className="btn btn-secondary">
                Read FAQ
              </a>
            </div>
          </div>
        </section>
      </main>

      <footer>
        <div className="wrap foot">
          <div>
            <a href="/" className="brand">
              <svg className="logo" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M2.5 5h19L12 20.5z" />
              </svg>
              Controller
            </a>
          </div>
          <div>
            <h4>Product</h4>
            <a href="/how-it-works">How it works</a>
            <a href="/faq">FAQ</a>
            <a href="/privacy">Privacy</a>
            <a href="/terms">Terms</a>
          </div>
          <div>
            <h4>Support</h4>
            <a href="/support">Help Center</a>
            <a href="/dashboard">Console</a>
          </div>
          <div>
            <h4>Legal</h4>
            <a href="/privacy">Privacy Policy</a>
            <a href="/terms">Terms of Service</a>
          </div>
        </div>
        <div className="wrap legal">
          <span>© {new Date().getFullYear()} Controller</span>
          <span>All rights reserved</span>
        </div>
      </footer>
    </>
  );
}
