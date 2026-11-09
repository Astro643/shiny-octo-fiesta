import Head from 'next/head';

export default function Terms() {
  return (
    <>
      <Head>
        <title>Terms of Service - Controller</title>
        <meta name="description" content="Terms of service for Controller." />
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
            <h1>Terms of Service</h1>
            <p className="updated">Last updated: June 2024</p>

            <div className="prose">
              <p>
                These Terms of Service (&quot;Terms&quot;) govern your use of Controller (&quot;Service&quot; or &quot;Controller&quot;). By using Controller, you agree to these Terms.
              </p>

              <h2>Use of Service</h2>
              <p>
                You must be at least 13 years old to use Controller. By using the Service, you represent and warrant that you are at least 13 years old.
              </p>
              <p>
                You must have the legal authority to manage the Discord server(s) you use Controller with. You represent that you are the owner or have explicit permission from the owner to use Controller on any server.
              </p>

              <h2>Prohibited Activities</h2>
              <p>
                You agree not to use Controller for any of the following purposes:
              </p>
              <ul>
                <li>Violating any applicable laws or regulations</li>
                <li>Violating Discord&apos;s Terms of Service</li>
                <li>Harassing, abusing, or harming others</li>
                <li>Spamming or sending unsolicited messages</li>
                <li>Distributing malware, viruses, or other harmful code</li>
                <li>Attempting to gain unauthorized access to any system</li>
                <li>Interfering with or disrupting the Service</li>
                <li>Using Controller for any commercial purposes without our permission</li>
              </ul>

              <h2>Your Responsibilities</h2>
              <p>
                You are solely responsible for all actions taken by Controller on your Discord server(s). This includes, but is not limited to:
              </p>
              <ul>
                <li>Any channels, roles, or messages created</li>
                <li>Any members kicked, banned, or timed out</li>
                <li>Any permissions changed</li>
                <li>Any data deleted</li>
              </ul>
              <p>
                We are not responsible for any damage, loss, or harm resulting from the use of Controller, including but not limited to loss of data, loss of server members, or damage to your reputation.
              </p>

              <h2>AI Safety</h2>
              <p>
                Controller uses AI to interpret your requests and execute actions. While we have implemented safety measures, the AI may not always interpret your requests correctly. You are responsible for reviewing the actions Controller plans to take before execution.
              </p>
              <p>
                The dry run preview feature is provided to help you understand what actions will be taken. However, it is your responsibility to ensure these actions are appropriate for your server.
              </p>

              <h2>Service Availability</h2>
              <p>
                We strive to provide a reliable service, but we do not guarantee that Controller will be available at all times. We may experience downtime for maintenance, updates, or other reasons.
              </p>
              <p>
                We are not liable for any damage, loss, or harm resulting from Service unavailability.
              </p>

              <h2>Termination</h2>
              <p>
                We may suspend or terminate your access to Controller at any time, with or without cause, and with or without notice.
              </p>
              <p>
                You may stop using Controller at any time by removing the bot from your server(s).
              </p>

              <h2>Disclaimer of Warranties</h2>
              <p>
                Controller is provided &quot;as is&quot; and &quot;as available&quot; without any warranties, express or implied. We disclaim all warranties, including but not limited to:
              </p>
              <ul>
                <li>Merchantability</li>
                <li>Fitness for a particular purpose</li>
                <li>Non-infringement</li>
                <li>Accuracy</li>
              </ul>

              <h2>Limitation of Liability</h2>
              <p>
                To the fullest extent permitted by law, we shall not be liable for any indirect, incidental, special, consequential, or punitive damages, including but not limited to loss of profits, data, use, or goodwill.
              </p>
              <p>
                Our total liability to you for any claim arising out of or related to these Terms or your use of Controller shall not exceed the greater of $100 or the amounts paid by you to us in the past 12 months.
              </p>

              <h2>Governing Law</h2>
              <p>
                These Terms shall be governed by and construed in accordance with the laws of the jurisdiction where the service operator resides, without regard to its conflict of law provisions.
              </p>

              <h2>Changes to These Terms</h2>
              <p>
                We may update these Terms from time to time. We will notify you of any changes by posting the new Terms on this page. Your continued use of Controller after such changes constitutes your acceptance of the new Terms.
              </p>

              <h2>Contact Us</h2>
              <p>
                If you have any questions about these Terms, please contact the service operator.
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
