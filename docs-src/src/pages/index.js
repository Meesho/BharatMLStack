import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import useBaseUrl from '@docusaurus/useBaseUrl';
import Layout from '@theme/Layout';
import { OnlineFeatureStoreFeatures, TruffleboxUIFeatures, SDKsFeatures } from '@site/src/components/HomepageFeatures';

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

function OnlineFeatureStoreAbout() {
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

function TruffleboxAbout() {
  return (
    <section className={styles.aboutSection}>
      <div className="container">
        <div className="row">
          <div className="col col--6">
            <Heading as="h2">Modern MLOps Management</Heading>
            <p>
              Trufflebox UI provides a comprehensive, modern web interface for managing your entire 
              ML infrastructure. Built with cutting-edge web technologies, it delivers an intuitive 
              experience for feature management, user administration, and operational oversight. 
              Streamline your MLOps workflows with enterprise-grade UI components.
            </p>
            <Link
              className="button button--primary"
              to="/category/trufflebox-ui">
              Explore Trufflebox UI →
            </Link>
          </div>
          <div className="col col--6">
            <div className={styles.highlightBox}>
              <h3>🎨 UI Features</h3>
              <ul>
                <li>✅ Comprehensive feature catalog & discovery</li>
                <li>✅ Role-based access control & user management</li>
                <li>✅ Job, Store, Admin Ops management</li>
                <li>✅ Approval flow for everything</li>
                <li>✅ Responsive design for desktop & mobile</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function SDKsAbout() {
  return (
    <section className={styles.aboutSection}>
      <div className="container">
        <div className="row">
          <div className="col col--6">
            <Heading as="h2">Developer-First Integration</Heading>
            <p>
              Our SDKs are designed with developers in mind, providing idiomatic APIs for Go and Python 
              that feel natural in your existing codebase. Whether you're building microservices, 
              data pipelines, or ML applications, our SDKs provide the tools you need for seamless 
              integration with BharatMLStack's powerful infrastructure.
            </p>
            <Link
              className="button button--primary"
              to="/category/sdks">
              Explore SDKs →
            </Link>
          </div>
          <div className="col col--6">
            <div className={styles.highlightBox}>
              <h3>🛠️ Developer Tools</h3>
              <ul>
                <li>✅ Native Go & Python SDKs with type safety</li>
                <li>✅ High-performance gRPC</li>
                <li>✅ Apache Spark integration for publishing features</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function NumerixAbout() {
  return (
    <section className={styles.aboutSection}>
      <div className="container">
        <div className="row">
          <div className="col col--6">
            <Heading as="h2">Numerix</Heading>
            <p>
              Numerix is a mathematical compute engine for BharatML Stack. It is used to perform mathematical operations on matrices and vectors.
            </p>
            <Link
              className="button button--primary"
              to="/category/numerix">
              Explore SDKs →
            </Link>
          </div>
          <div className="col col--6">
            <div className={styles.highlightBox}>
              <h3>🛠️ Numerix Features</h3>
              <ul>
                <li>✅ Postfix expression evaluation</li>
                <li>✅ Vectorized math operations</li>
                <li>✅ Typed evaluation</li>
                <li>✅ Compiler-assisted SIMD</li>
                <li>✅ ARM & AMD support</li>
                <li>✅ Multi-arch builds</li>
                <li>✅ Deterministic runtime</li>

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
        <OnlineFeatureStoreFeatures />
        <OnlineFeatureStoreAbout />
        <TruffleboxUIFeatures />
        <TruffleboxAbout />
        <SDKsFeatures />
        <SDKsAbout />
        <NumerixAbout />
      </main>
    </Layout>
  );
}
