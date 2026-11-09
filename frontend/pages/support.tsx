import Head from 'next/head';

export default function Support() {
  return (
    <>
      <Head>
        <title>Support - Controller</title>
        <meta name="description" content="Get support for Controller. AI-powered Discord management." />
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
              <a href="/support" className="on">Support</a>
            </nav>
            <a href="/dashboard" className="btn btn-primary">
              Open Console
            </a>
          </nav>
        </div>
      </header>

      <main>
        <section className="page">
          <div className="wrap">
            <span className="eyebrow">Support</span>
            <h1>Help Center</h1>
            <p className="lead">Get help with Controller. We&apos;re here to assist you.</p>

            <div className="prose">
              <h2>Getting Started</h2>
              <p>
                New to Controller? Check out our <a href="/how-it-works">How It Works</a> page and <a href="/faq">FAQ</a> for answers to common questions.
              </p>

              <h2>Common Issues</h2>
              <p>If you&apos;re experiencing issues:</p>
              <ul>
                <li><strong>Session expired:</strong> Open a fresh dashboard link from your bot by running <code>/dashboard</code> again.</li>
                <li><strong>Agent unavailable:</strong> All agents might be busy. Wait a few minutes and try again.</li>
                <li><strong>Request failed:</strong> Check the error message in your console. It usually explains what went wrong.</li>
                <li><strong>Bot not responding:</strong> Make sure the bot has the necessary permissions and is online in your server.</li>
              </ul>

              <h2>Permissions</h2>
              <p>
                The Controller bot needs the following permissions to work properly:
              </p>
              <ul>
                <li>Administrator (recommended for full functionality)</li>
                <li>Or specific permissions: Manage Channels, Manage Roles, Kick Members, Ban Members, etc.</li>
              </ul>
              <p>
                The bot&apos;s role should be <strong>above</strong> the roles of any members it needs to moderate.
              </p>

              <h2>System Status</h2>
              <p>
                You can check the live system status at any time. If there are any issues with our services, they will be displayed there.
              </p>

              <h2>Contact Us</h2>
              <p>
                If you can&apos;t find the answer you&apos;re looking for, you can reach out to us:
              </p>
              <ul>
                <li>Join our support Discord server (link available when service is configured)</li>
                <li>Send us an email at the address configured by the operator</li>
              </ul>
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
