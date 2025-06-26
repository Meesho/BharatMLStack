import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

const FeatureList = [
  {
    title: 'High-Performance Feature Store',
    icon: '🚀',
    description: (
      <>
        Sub-10ms P99 latency and 1M+ RPS capacity. Built for real-time ML inference 
        with custom PSDB serialization format that outperforms Protocol Buffers and Apache Arrow.
      </>
    ),
  },
  {
    title: 'Production-Ready ML Infrastructure',
    icon: '⚡',
    description: (
      <>
        Multi-database backends (Scylla, Dragonfly, Redis), comprehensive monitoring, 
        and enterprise-grade features. Deploy with confidence using battle-tested components.
      </>
    ),
  },
  {
    title: 'Developer-First Experience',
    icon: '🛠️',
    description: (
      <>
        Multi-language SDKs (Go, Python), gRPC APIs, and extensive documentation. 
        From data scientists, ML engineers to backend engineers, everyone gets tools they love.                    
      </>
    ),
  },
];

const TruffleboxFeatures = [
  {
    title: 'Feature Catalog & Management',
    icon: '📋',
    description: (
      <>
        Comprehensive feature catalog with metadata management, versioning, and governance. 
        Organize and discover features across your ML platform with ease.
      </>
    ),
  },
  {
    title: 'User Management & Admin Ops',
    icon: '👥',
    description: (
      <>
        Role-based access control, user authentication, and administrative operations. 
        Secure your ML platform with enterprise-grade user management capabilities.
      </>
    ),
  },
  {
    title: 'Modern UI Framework',
    icon: '🎨',
    description: (
      <>
        Intuitive, responsive web interface built with modern web technologies. 
        Streamline MLOps workflows with beautiful and functional user experiences.
      </>
    ),
  },
];

const SDKFeatures = [
  {
    title: 'Multi-Language Support',
    icon: '🌐',
    description: (
      <>
        Native SDKs for Go and Python with idiomatic APIs. Choose the language that fits 
        your team's expertise and existing infrastructure.
      </>
    ),
  },
  {
    title: 'gRPC & REST APIs',
    icon: '🔗',
    description: (
      <>
        High-performance gRPC clients and REST APIs for seamless integration. 
        Built-in support for streaming, batching, and async operations.
      </>
    ),
  },
  {
    title: 'Spark Integration',
    icon: '⚡',
    description: (
      <>
        Native Apache Spark integration for batch feature processing and ingestion. 
        Scale your feature engineering workflows with distributed computing power.
      </>
    ),
  },
];

function Feature({icon, title, description}) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <div className="bharatml-icon">
          {icon}
        </div>
      </div>
      <div className="text--center padding-horiz--md bharatml-card">
        <Heading as="h3">{title}</Heading>
        <p className={styles.featureDescription}>{description}</p>
      </div>
    </div>
  );
}

function FeatureSection({title, subtitle, features}) {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="text--center margin-bottom--xl">
          <Heading as="h2" className={styles.featuresHeader}>
            {title}
          </Heading>
          <p className={styles.featuresSubtitle}>
            {subtitle}
          </p>
        </div>
        <div className="row">
          {features.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}

export function OnlineFeatureStoreFeatures() {
  return (
    <FeatureSection
      title="Online Feature Store"
      subtitle="High-performance, production-ready feature serving for real-time ML inference"
      features={FeatureList}
    />
  );
}

export function TruffleboxUIFeatures() {
  return (
    <FeatureSection
      title="Trufflebox UI"
      subtitle="Modern, feature-rich UI framework for comprehensive MLOps management"
      features={TruffleboxFeatures}
    />
  );
}

export function SDKsFeatures() {
  return (
    <FeatureSection
      title="SDKs"
      subtitle="Developer-friendly client libraries and APIs for seamless platform integration"
      features={SDKFeatures}
    />
  );
}

// Default export for backward compatibility
export default function HomepageFeatures() {
  return (
    <>
      <OnlineFeatureStoreFeatures />
      <TruffleboxUIFeatures />
      <SDKsFeatures />
    </>
  );
}
