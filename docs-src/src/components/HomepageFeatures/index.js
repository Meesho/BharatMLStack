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

export default function HomepageFeatures() {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="text--center margin-bottom--xl">
          <Heading as="h2" className={styles.featuresHeader}>
            Online Feature Store
          </Heading>
          <p className={styles.featuresSubtitle}>
            High-performance, production-ready feature serving for real-time ML inference
          </p>
        </div>
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
