import Head from 'next/head';

export default function Privacy() {
  return (
    <>
      <Head>
        <title>Privacy Policy - Controller</title>
        <meta name="description" content="Privacy policy for Controller." />
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
        <section className="page">
          <div className="wrap">
            <h1>Privacy Policy</h1>
            <p className="updated">Last updated: June 2024</p>

            <div className="prose">
              <p>
                This Privacy Policy describes how Controller (&quot;we,&quot; &quot;us,&quot; or &quot;our&quot;) collects, uses, and discloses your information when you use our service.
              </p>

              <h2>Information We Collect</h2>
              <p>
                We collect information you provide directly to us, such as when you use the Controller console to submit requests for your Discord server.
              </p>

              <h3>Server Information</h3>
              <p>
                When you use Controller, we collect information about your Discord server, including:
              </p>
              <ul>
                <li>Guild ID</li>
                <li>Guild name</li>
                <li>Channel names and IDs</li>
                <li>Role names and IDs</li>
                <li>Member names and IDs (for moderation actions)</li>
              </ul>

              <h3>Request Information</h3>
              <p>
                We collect the text of your requests and the actions taken by the bot, including:
              </p>
              <ul>
                <li>The prompt/command you submit</li>
                <li>The AI&apos;s response and planned actions</li>
                <li>The final status of each request</li>
                <li>Any errors that occur</li>
              </ul>

              <h2>How We Use Your Information</h2>
              <p>
                We use the information we collect to:
              </p>
              <ul>
                <li>Provide, maintain, and improve our service</li>
                <li>Process your requests and execute the actions you command</li>
                <li>Monitor and analyze usage and trends</li>
                <li>Prevent abuse and harmful actions</li>
                <li>Comply with legal obligations</li>
              </ul>

              <h2>AI Processing</h2>
              <p>
                Your requests and server context are sent to Google&apos;s Gemini API for processing. This is necessary for the AI to understand your requests and generate appropriate responses. Google&apos;s privacy policy applies to this data processing.
              </p>

              <h2>Data Retention</h2>
              <p>
                We retain your request history and server snapshots for up to 30 days. After this period, the data is automatically deleted. Irreversible actions (like bans and kicks) are logged in an audit log for the same period.
              </p>
              <p>
                Session data (like your authentication token) is retained for 12 hours or until you close your browser, whichever comes first.
              </p>

              <h2>Data Sharing</h2>
              <p>
                We do not sell, trade, or otherwise transfer your personal information to third parties, except as described in this policy or with your consent.
              </p>
              <p>
                We may share information with:
              </p>
              <ul>
                <li>Google (for AI processing via Gemini API)</li>
                <li>Discord (to execute actions on your server)</li>
                <li>Legal authorities (if required by law)</li>
              </ul>

              <h2>Your Rights</h2>
              <p>
                Depending on your location, you may have certain rights regarding your data, including:
              </p>
              <ul>
                <li>The right to access your data</li>
                <li>The right to request deletion of your data</li>
                <li>The right to correct inaccurate data</li>
              </ul>
              <p>
                To exercise these rights, please contact the service operator.
              </p>

              <h2>Children&apos;s Privacy</h2>
              <p>
                Our service is not directed to children under the age of 13. We do not knowingly collect personal information from children under 13.
              </p>

              <h2>Changes to This Policy</h2>
              <p>
                We may update this Privacy Policy from time to time. We will notify you of any changes by posting the new policy on this page.
              </p>

              <h2>Contact Us</h2>
              <p>
                If you have any questions about this Privacy Policy, please contact the service operator.
              </p>
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
