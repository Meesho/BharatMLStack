import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import useBaseUrl from '@docusaurus/useBaseUrl';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';

import Heading from '@theme/Heading';
import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero bharatml-hero', styles.heroBanner)}>
      <div className="container">
        <div className={styles.logoContainer}>
          <img 
            src={useBaseUrl('/img/logo.svg')} 
            alt="BharatMLStack Logo" 
            className={styles.heroLogo}
          />
        </div>
        <Heading as="h1" className="hero__title">
          Welcome to {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">
          Open source, end-to-end ML infrastructure stack built for scale, speed, and simplicity.
        </p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg margin-right--md bharatml-button"
            to="/category/online-feature-store">
            📚 Get Started
          </Link>
          <Link
            className="button button--outline button--secondary button--lg"
            href="https://github.com/Meesho/BharatMLStack"
            target="_blank">
            ⭐ Star on GitHub
          </Link>
        </div>
        <div className={styles.statsContainer}>
          <div className={styles.statItem}>
            <strong>Sub-10ms</strong>
            <span>P99 Latency</span>
          </div>
          <div className={styles.statItem}>
            <strong>1M+ RPS</strong>
            <span>Tested Capacity</span>
          </div>
          <div className={styles.statItem}>
            <strong>Multi-DB</strong>
            <span>Support</span>
          </div>
        </div>
      </div>
    </header>
  );
}

function AboutSection() {
  return (
    <section className={styles.aboutSection}>
      <div className="container">
        <div className="row">
          <div className="col col--6">
            <Heading as="h2">Built for India's Scale</Heading>
            <p>
              BharatMLStack is a comprehensive, production-ready machine learning infrastructure 
              platform designed to democratize ML capabilities across India and beyond. Our mission 
              is to provide a robust, scalable, and accessible ML stack that empowers organizations 
              to build, deploy, and manage machine learning solutions at massive scale.
            </p>
            <Link
              className="button button--primary"
              to="/category/online-feature-store">
              Explore Online Feature Store →
            </Link>
          </div>
          <div className="col col--6">
            <div className={styles.highlightBox}>
              <h3>🏆 Key Achievements</h3>
              <ul>
                <li>✅ Sub-10ms P99 latency for real-time inference</li>
                <li>✅ 1M+ RPS tested with 100 IDs per request</li>
                <li>✅ PSDB format outperforms Proto3 & Arrow</li>
                <li>✅ Multi-database: Scylla, Dragonfly, Redis</li>
                <li>✅ Production-ready with comprehensive monitoring</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

export default function Home() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title} - Open Source ML Infrastructure`}
      description="Open source, end-to-end ML infrastructure stack built for scale, speed, and simplicity. Features high-performance Online Feature Store with sub-10ms latency.">
      <HomepageHeader />
      <main>
        <HomepageFeatures />
        <AboutSection />
      </main>
    </Layout>
  );
}
