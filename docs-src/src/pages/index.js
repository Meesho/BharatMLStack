import React, { useEffect, useLayoutEffect, useRef, useState, useCallback } from 'react';
import Layout from '@theme/Layout';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import useBaseUrl from '@docusaurus/useBaseUrl';
import styles from './index.module.css';

// ─── Data ──────────────────────────────────────────────

const BARRIERS = [
  {
    icon: '\u{1F9E0}',
    title: 'Focus on building intelligence, not infrastructure',
    questions: [
      'Does every model deployment require a full-stack integration effort?',
      'Do engineers have to rebuild feature retrieval, endpoint integrations, and logging for each new model?',
      'Does changing a simple expression like 0.2\u00D7s\u2081 + 0.8\u00D7s\u2082 to 0.3\u00D7s\u2081 + 0.7\u00D7s\u2082 really need code reviews and redeployments?',
      'Why does deploying intelligence require the devops team to provision infra?',
    ],
    answer:
      'Machine learning teams should be iterating on models, not systems. Yet today, infrastructure complexity turns simple improvements into weeks of engineering effort, slowing experimentation and innovation.',
  },
  {
    icon: '\u{1F4B0}',
    title: 'Built for scale without exponential cost growth',
    questions: [
      'Do your infrastructure costs scale faster than your ML impact?',
      'Are you recomputing the same features, reloading the same data, and moving the same bytes across systems repeatedly?',
      'Are expensive GPUs and compute sitting underutilized while workloads wait on data or inefficient pipelines?',
      'Why does scaling ML often mean scaling cost linearly\u2014or worse?',
    ],
    answer:
      'A modern ML platform should eliminate redundant computation, reuse features intelligently, and optimize data access across memory, NVMe, and object storage. Compute should be pooled, scheduled efficiently, and fully utilized\u2014ensuring that scale drives impact, not runaway infrastructure costs.',
  },
  {
    icon: '\u{1F30D}',
    title: 'Freedom to deploy anywhere, without lock-in',
    questions: [
      'Are your models tied to a single cloud, making migration costly and complex?',
      'Does adopting managed services today limit your ability to optimize cost or move infrastructure tomorrow?',
      'Can you deploy the same ML stack across public cloud, private cloud, or sovereign environments without redesigning everything?',
      'Why should infrastructure choices dictate the future of your ML systems?',
    ],
    answer:
      'A modern ML platform should be built on open standards and cloud-neutral abstractions, allowing you to deploy anywhere\u2014public cloud, private infrastructure, or sovereign environments. This ensures complete control over your data, freedom from vendor lock-in, and the ability to optimize for cost, performance, and compliance without architectural constraints.',
  },
];

const COMPONENTS = [
  {
    icon: '\u{26A1}',
    title: 'Online Feature Store',
    description:
      'BharatMLStack Online Feature Store delivers sub-10ms, high-throughput access to machine learning features for real-time inference. It seamlessly ingests batch and streaming data, validates schemas, and persists compact, versioned feature groups optimized for low latency and efficiency. With scalable storage backends, gRPC APIs, and binary-optimized formats, it ensures consistent, reliable feature serving across ML pipelines.',
    cta: '/online-feature-store/v1.0.0',
  },
  {
    icon: '\u{1F500}',
    title: 'Inferflow',
    description:
      "Inferflow is BharatMLStack's intelligent inference gateway that dynamically retrieves and assembles features required by ML models using a graph-based configuration called Inferpipes. It automatically resolves entity relationships, fetches features from the Online Feature Store, and constructs feature vectors without custom code.",
    cta: '/inferflow/v1.0.0',
  },
  {
    icon: '\u{1F50D}',
    title: 'Skye',
    description:
      'Skye enables fast similarity retrieval by representing data as vectors and querying nearest matches in high-dimensional space. It supports pluggable vector databases, ensuring flexibility across infrastructure. The system provides tenant-level index isolation while allowing single embedding ingestion even when shared across tenants, reducing redundancy.',
    cta: '/skye/v1.0.0',
  },
  {
    icon: '\u{1F9EE}',
    title: 'Numerix',
    description:
      'Numerix is a high-performance compute engine designed for ultra-fast element-wise matrix operations. Built in Rust and accelerated using SIMD, it delivers exceptional efficiency and predictable performance. Optimized for real-time inference workloads, it achieves strict sub-5ms p99 latency on matrices up to 1000\u00D710.',
    cta: '/numerix/v1.0.0',
  },
  {
    icon: '\u{1F680}',
    title: 'Predator',
    description:
      'Predator streamlines infrastructure and model lifecycle management. It enables the creation of deployables with specific Triton Server versions and supports seamless model rollouts. Leveraging Helm charts and Argo CD, Predator automates Kubernetes-based deployments while integrating with KEDA for auto-scaling and performance tuning.',
    cta: '/predator/v1.0.0',
  },
];

const STATS = [
  { target: 4.5, suffix: 'M+', decimals: 1, label: 'Daily Orders', description: 'Daily orders processed via ML pipelines' },
  { target: 2.4, suffix: 'M', decimals: 1, label: 'QPS on FS', description: 'QPS on Feature Store with batch size of 100 id lookups' },
  { target: 1, suffix: 'M+', decimals: 0, label: 'QPS Inference', description: 'QPS on Model Inference' },
  { target: 500, suffix: 'K', decimals: 0, label: 'QPS Embedding', description: 'QPS Embedding Search' },
];

const DEMO_VIDEOS = [
  {
    title: 'Feature Store',
    description: 'Learn how to onboard and manage features using the self-serve UI for the Online Feature Store.',
    url: 'https://videos.meesho.com/reels/feature_store.mp4',
  },
  {
    title: 'Embedding Platform',
    description: 'Walkthrough of onboarding and managing embedding models via the Skye self-serve UI.',
    url: 'https://videos.meesho.com/reels/embedding_platform.mp4',
  },
  {
    title: 'Numerix',
    description: 'Step-by-step guide to configuring and running matrix operations through the Numerix self-serve UI.',
    url: 'https://videos.meesho.com/reels/numerix.mp4',
  },
  {
    title: 'Predator',
    description: 'How to deploy and manage ML models on Kubernetes using the Predator self-serve UI.',
    url: 'https://videos.meesho.com/reels/predator.mp4',
  },
  {
    title: 'Inferflow',
    description: 'Setting up inferpipes and feature retrieval graphs through the Inferflow self-serve UI.',
    url: 'https://videos.meesho.com/reels/inferflow.mp4',
  },
];

const BLOG_POSTS = [
  {
    title: "Building Meesho's ML Platform: From Chaos to Cutting-Edge (Part 1)",
    category: 'ML Platform',
    icon: '\u{1F680}',
    link: '/blog/post-one',
  },
  {
    title: "Building Meesho's ML Platform: Lessons from the First-Gen System (Part 2)",
    category: 'ML Platform',
    icon: '\u{1F9E9}',
    link: '/blog/post-two',
  },
  {
    title: 'Cracking the Code: Scaling Model Inference & Real-Time Embedding Search',
    category: 'Inference',
    icon: '\u{26A1}',
    link: '/blog/post-three',
  },
  {
    title: 'Designing a Production-Grade LLM Inference Platform: From Model Weights to Scalable GPU Serving',
    category: 'LLM',
    icon: '\u{1F9E0}',
    link: '/blog/post-four',
  },
  {
    title: 'LLM Inference Optimization Techniques: Engineering Sub-Second Latency at Scale',
    category: 'Optimization',
    icon: '\u{1F52C}',
    link: '/blog/post-five',
  },
  {
    title: "Beyond Vector RAG: Building Agent Memory That Learns From Experience.",
    category: 'AI Agents',
    icon: '\u{1F9E0}',
    link: '/blog/episodic-memory-for-agents',
  },
];

// ─── Components ────────────────────────────────────────

function CustomNav() {
  const docsUrl = useBaseUrl('/');
  const blogUrl = useBaseUrl('/blog');
  return (
    <nav className={styles.customNav}>
      <div className={styles.navContainer}>
        <a href={docsUrl} className={styles.logo}>BharatMLStack</a>
        <div className={styles.navLinks}>
          <a href="#components" className={styles.navLink}>Components</a>
          <a href="#stats" className={styles.navLink}>Scale</a>
          <a href="#demos" className={styles.navLink}>Demos</a>
          <a href={blogUrl} className={styles.navLink}>Blog</a>
          <a
            href="https://github.com/Meesho/BharatMLStack"
            className={`${styles.btn} ${styles.btnPrimary}`}
            target="_blank"
            rel="noopener noreferrer"
          >
            GitHub
          </a>
        </div>
      </div>
    </nav>
  );
}

function NetworkBackground() {
  const canvasRef = useRef(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    let animationId;
    let nodes = [];
    const NODE_COUNT = 50;
    const CONNECTION_DIST = 150;

    function resize() {
      const parent = canvas.parentElement;
      canvas.width = parent.offsetWidth;
      canvas.height = parent.offsetHeight;
    }

    function initNodes() {
      nodes = [];
      for (let i = 0; i < NODE_COUNT; i++) {
        nodes.push({
          x: Math.random() * canvas.width,
          y: Math.random() * canvas.height,
          vx: (Math.random() - 0.5) * 0.4,
          vy: (Math.random() - 0.5) * 0.4,
          radius: Math.random() * 2 + 1,
        });
      }
    }

    function draw() {
      ctx.clearRect(0, 0, canvas.width, canvas.height);

      // Draw connections
      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const dx = nodes[i].x - nodes[j].x;
          const dy = nodes[i].y - nodes[j].y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          if (dist < CONNECTION_DIST) {
            const opacity = (1 - dist / CONNECTION_DIST) * 0.25;
            ctx.beginPath();
            ctx.moveTo(nodes[i].x, nodes[i].y);
            ctx.lineTo(nodes[j].x, nodes[j].y);
            ctx.strokeStyle = `rgba(99, 102, 241, ${opacity})`;
            ctx.lineWidth = 0.8;
            ctx.stroke();
          }
        }
      }

      // Draw nodes
      for (const node of nodes) {
        ctx.beginPath();
        ctx.arc(node.x, node.y, node.radius, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(139, 92, 246, 0.5)';
        ctx.fill();
      }
    }

    function update() {
      for (const node of nodes) {
        node.x += node.vx;
        node.y += node.vy;
        // Bounce off edges
        if (node.x < 0 || node.x > canvas.width) node.vx *= -1;
        if (node.y < 0 || node.y > canvas.height) node.vy *= -1;
        // Keep in bounds
        node.x = Math.max(0, Math.min(canvas.width, node.x));
        node.y = Math.max(0, Math.min(canvas.height, node.y));
      }
    }

    function animate() {
      update();
      draw();
      animationId = requestAnimationFrame(animate);
    }

    resize();
    initNodes();
    animate();

    const resizeObserver = new ResizeObserver(() => {
      resize();
    });
    resizeObserver.observe(canvas.parentElement);

    return () => {
      cancelAnimationFrame(animationId);
      resizeObserver.disconnect();
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className={styles.networkCanvas}
      aria-hidden="true"
    />
  );
}

function HeroSection() {
  const getStartedUrl = useBaseUrl('/intro');
  return (
    <section className={styles.hero}>
      <NetworkBackground />
      <div className={styles.heroContent}>
        <div className={styles.heroBadge}>Open-source, scalable stack for enterprise ML</div>
        <h1 className={styles.heroTitle}>Build production ML pipelines faster</h1>
        <p className={styles.heroSubtitle}>
          Open source, end-to-end ML infrastructure stack built for scale, speed, and simplicity.
          Integrate, deploy, and manage robust ML workflows with full reliability and control.
        </p>
        <div className={styles.heroButtons}>
          <a href={getStartedUrl} className={`${styles.btn} ${styles.btnPrimary}`}>
            Get Started
          </a>
          <a
            href="https://github.com/Meesho/BharatMLStack"
            className={`${styles.btn} ${styles.btnSecondary}`}
            target="_blank"
            rel="noopener noreferrer"
          >
            View on GitHub
          </a>
        </div>
        <div className={styles.adoptionBadge}>
          <p>Adopted by data teams building at scale</p>
        </div>
      </div>
      <div className={styles.heroImage}>
        <img
          src={useBaseUrl('/img/bharatml-stack-logo.jpg')}
          alt="BharatML Stack Logo"
          loading="eager"
        />
      </div>
    </section>
  );
}

function BarriersSection() {
  return (
    <section className={styles.section}>
      <div className={styles.container}>
        <div className={styles.sectionHeader}>
          <p className={styles.sectionSubtitle}>Why BharatMLStack</p>
          <h2 className={styles.sectionTitle}>The Real Barriers to Scaling Machine Learning</h2>
          <p className={styles.sectionDescription}>
            ML teams spend more time fighting infrastructure than building intelligence.
            BharatMLStack removes those barriers.
          </p>
        </div>
        <div className={styles.barriersGrid}>
          {BARRIERS.map((barrier, idx) => (
            <div className={styles.barrierCard} key={idx}>
              <div className={styles.barrierIcon}>{barrier.icon}</div>
              <h3>{barrier.title}</h3>
              <ul className={styles.barrierQuestions}>
                {barrier.questions.map((q, i) => (
                  <li key={i}>{q}</li>
                ))}
              </ul>
              <p className={styles.barrierAnswer}>{barrier.answer}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function ComponentsSection() {
  const cardsRef = useRef([]);
  const baseUrl = useBaseUrl('/');

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add(styles.componentCardVisible);
          }
        });
      },
      { threshold: 0.1, rootMargin: '0px 0px -80px 0px' }
    );

    cardsRef.current.forEach((card) => {
      if (card) observer.observe(card);
    });

    return () => observer.disconnect();
  }, []);

  return (
    <section className={styles.section} id="components">
      <div className={styles.container}>
        <div className={styles.sectionHeader}>
          <p className={styles.sectionSubtitle}>Platform Components</p>
          <h2 className={styles.sectionTitle}>BharatMLStack Components</h2>
          <p className={styles.sectionDescription}>
            Purpose-built components for every stage of the ML lifecycle, from feature
            serving to model deployment.
          </p>
        </div>
        <div className={styles.componentsGrid}>
          {COMPONENTS.map((comp, idx) => (
            <div
              className={styles.componentCard}
              key={idx}
              ref={(el) => (cardsRef.current[idx] = el)}
            >
              <div className={styles.componentIcon}>{comp.icon}</div>
              <div className={styles.componentContent}>
                <h3>{comp.title}</h3>
                <p>{comp.description}</p>
                <a href={`${baseUrl}${comp.cta.replace(/^\//, '')}`} className={styles.componentLink}>
                  Learn more &rarr;
                </a>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function AnimatedCounter({ target, suffix, decimals, duration = 1500 }) {
  const [count, setCount] = useState(0);
  const [hasStarted, setHasStarted] = useState(false);
  const ref = useRef(null);

  const startAnimation = useCallback(() => {
    if (hasStarted) return;
    setHasStarted(true);

    const startTime = performance.now();
    const step = (now) => {
      const elapsed = now - startTime;
      const progress = Math.min(elapsed / duration, 1);
      // Ease-out cubic for a fast start that decelerates
      const eased = 1 - Math.pow(1 - progress, 3);
      setCount(eased * target);
      if (progress < 1) {
        requestAnimationFrame(step);
      } else {
        setCount(target);
      }
    };
    requestAnimationFrame(step);
  }, [target, duration, hasStarted]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          startAnimation();
        }
      },
      { threshold: 0.3 }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [startAnimation]);

  const display = decimals > 0
    ? count.toFixed(decimals)
    : Math.round(count).toLocaleString();

  return (
    <div className={styles.statValue} ref={ref}>
      {display}{suffix}
    </div>
  );
}

function StatsSection() {
  return (
    <section className={`${styles.section} ${styles.statsSection}`} id="stats">
      <div className={styles.container}>
        <div className={styles.sectionHeader}>
          <p className={styles.sectionSubtitle}>Proven at scale</p>
          <h2 className={styles.sectionTitle}>Scaling Numbers</h2>
        </div>
        <div className={styles.statsGrid}>
          {STATS.map((stat, idx) => (
            <div className={styles.statCard} key={idx}>
              <p className={styles.statLabel}>{stat.label}</p>
              <AnimatedCounter
                target={stat.target}
                suffix={stat.suffix}
                decimals={stat.decimals}
              />
              <p className={styles.statDescription}>{stat.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function DemoVideosSection() {
  return (
    <section className={styles.section} id="demos">
      <div className={styles.container}>
        <div className={styles.sectionHeader}>
          <p className={styles.sectionSubtitle}>See it in action</p>
          <h2 className={styles.sectionTitle}>Demo Videos</h2>
          <p className={styles.sectionDescription}>
            Watch short demos of each BharatMLStack component in action.
          </p>
        </div>
        <div className={styles.videosGrid}>
          {DEMO_VIDEOS.map((video, idx) => (
            <div className={styles.videoCard} key={idx}>
              <div className={styles.videoWrapper}>
                <video
                  className={styles.videoPlayer}
                  controls
                  preload="metadata"
                  playsInline
                >
                  <source src={video.url} type="video/mp4" />
                  Your browser does not support the video tag.
                </video>
              </div>
              <div className={styles.videoContent}>
                <h3>{video.title}</h3>
                <p>{video.description}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function BlogSection() {
  const baseUrl = useBaseUrl('/');
  return (
    <section className={styles.section} id="blog">
      <div className={styles.container}>
        <div className={styles.sectionHeader}>
          <p className={styles.sectionSubtitle}>From our blog</p>
          <h2 className={styles.sectionTitle}>View Our Blogs</h2>
          <p className={styles.sectionDescription}>
            Technical articles, architecture deep-dives, and the story behind BharatMLStack.
          </p>
        </div>
        <div className={styles.blogGrid}>
          {BLOG_POSTS.map((post, idx) => (
            <a href={`${baseUrl}${post.link.replace(/^\//, '')}`} className={styles.blogCard} key={idx}>
              <div className={styles.blogCardIcon}>{post.icon}</div>
              <div className={styles.blogContent}>
                <span className={styles.blogCategory}>{post.category}</span>
                <h3>{post.title}</h3>
                <div className={styles.blogMeta}>
                  <span>BharatMLStack Team</span>
                </div>
              </div>
            </a>
          ))}
        </div>
      </div>
    </section>
  );
}

function CTASection() {
  const getStartedUrl = useBaseUrl('/intro');
  return (
    <section className={styles.section}>
      <div className={styles.container}>
        <div className={styles.ctaSection}>
          <h2 className={styles.ctaTitle}>Deploy ML models with confidence</h2>
          <p className={styles.ctaDescription}>
            Comprehensive stack for business-ready ML. Integrates seamlessly with enterprise
            systems. Robust security and regulatory compliance.
          </p>
          <div className={styles.ctaButtons}>
            <a href={getStartedUrl} className={`${styles.btn} ${styles.btnWhite}`}>
              Start Now
            </a>
            <a
              href="https://github.com/Meesho/BharatMLStack"
              className={`${styles.btn} ${styles.btnOutlineWhite}`}
              target="_blank"
              rel="noopener noreferrer"
            >
              View on GitHub
            </a>
          </div>
        </div>
      </div>
    </section>
  );
}

function CustomFooter() {
  const docsUrl = useBaseUrl('/');
  const blogUrl = useBaseUrl('/blog');
  return (
    <footer className={styles.customFooter}>
      <div className={styles.footerContent}>
        <div className={styles.footerSection}>
          <h4>BharatMLStack</h4>
          <p>
            Enterprise-ready open-source ML infrastructure built for scale, speed, and
            simplicity.
          </p>
        </div>

        <div className={styles.footerSection}>
          <h4>Platform</h4>
          <ul className={styles.footerList}>
            <li><a href={useBaseUrl('/online-feature-store/v1.0.0')}>Online Feature Store</a></li>
            <li><a href={useBaseUrl('/inferflow/v1.0.0')}>Inferflow</a></li>
            <li><a href={useBaseUrl('/skye/v1.0.0')}>Skye</a></li>
            <li><a href={useBaseUrl('/numerix/v1.0.0')}>Numerix</a></li>
            <li><a href={useBaseUrl('/predator/v1.0.0')}>Predator</a></li>
          </ul>
        </div>

        <div className={styles.footerSection}>
          <h4>Resources</h4>
          <ul className={styles.footerList}>
            <li><a href={blogUrl}>Blog</a></li>
            <li><a href={docsUrl}>Documentation</a></li>
            <li><a href="https://github.com/Meesho/BharatMLStack/discussions">Forum</a></li>
          </ul>
        </div>

        <div className={styles.footerSection}>
          <h4>Community</h4>
          <ul className={styles.footerList}>
            <li><a href="https://github.com/Meesho/BharatMLStack">GitHub</a></li>
            <li><a href="https://discord.gg/XkT7XsV2AU">Discord</a></li>
            <li><a href="https://github.com/Meesho/BharatMLStack/blob/main/CONTRIBUTING.md">Contributing</a></li>
          </ul>
        </div>
      </div>

      <div className={styles.footerBottom}>
        <p>&copy; {new Date().getFullYear()} Meesho Ltd. All rights reserved. Open Source under Apache 2.0 License.</p>
        <div className={styles.footerLinks}>
          <a href="https://github.com/Meesho/BharatMLStack">GitHub</a>
        </div>
      </div>
    </footer>
  );
}

// ─── Page ──────────────────────────────────────────────

export default function Home() {
  const { siteConfig } = useDocusaurusContext();

  // Hide Docusaurus navbar/footer on homepage (client-side, before paint)
  useLayoutEffect(() => {
    document.documentElement.classList.add('homepage-active');
    return () => {
      document.documentElement.classList.remove('homepage-active');
    };
  }, []);

  return (
    <Layout
      title={`${siteConfig.title} - Open Source ML Infrastructure`}
      description="Open source, end-to-end ML infrastructure stack built for scale, speed, and simplicity."
    >
      {/* Inline style ensures Docusaurus navbar/footer are hidden during SSR and before JS hydration */}
      <style>{`
        .navbar { display: none !important; }
        .footer { display: none !important; }
        [class*='docMainContainer'], [class*='mainWrapper'] { padding-top: 0 !important; }
        main { margin-top: 0 !important; }
      `}</style>
      <div className={styles.homepageWrapper}>
        <CustomNav />
        <HeroSection />
        <BarriersSection />
        <ComponentsSection />
        <StatsSection />
        <DemoVideosSection />
        <BlogSection />
        <CTASection />
        <CustomFooter />
      </div>
    </Layout>
  );
}
