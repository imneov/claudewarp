import React from 'react';
import clsx from 'clsx';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<'svg'>>;
  description: React.ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'All-in-One Remote Control',
    Svg: require('@site/static/img/terminal.svg').default,
    description: (
      <>
        Built-in web interface for seamless Claude interaction. 
        PTY technology ensures complete I/O capture without complexity.
      </>
    ),
  },
  {
    title: 'Real-time Collaboration',
    Svg: require('@site/static/img/monitor.svg').default,
    description: (
      <>
        WebSocket-powered multi-user monitoring with session history.
        Team collaboration made simple and efficient.
      </>
    ),
  },
  {
    title: 'Multi-platform Bridge',
    Svg: require('@site/static/img/interaction.svg').default,
    description: (
      <>
        Native integrations for Telegram, Slack, Discord and more.
        Deploy once, integrate everywhere with minimal configuration.
      </>
    ),
  },
];

function Feature({title, Svg, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="feature-card">
        <div className="text--center">
          <Svg className={styles.featureSvg} role="img" />
        </div>
        <div className="text--center">
          <h3>{title}</h3>
          <p>{description}</p>
        </div>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): React.ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          <div className="col col--8 col--offset-2">
            <div style={{ textAlign: 'center', marginBottom: '3rem' }}>
              <h2 style={{ 
                fontSize: '2rem', 
                fontWeight: 600, 
                marginBottom: '1rem',
                color: '#202124'
              }}>
                Why ClaudeWarp?
              </h2>
              <p style={{ 
                fontSize: '1.1rem', 
                color: '#5f6368', 
                lineHeight: 1.6,
                maxWidth: '600px',
                margin: '0 auto'
              }}>
                You don't have to rebuild everything from scratch. ClaudeWarp provides all the tools you need to control and monitor Claude effectively.
              </p>
            </div>
          </div>
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
