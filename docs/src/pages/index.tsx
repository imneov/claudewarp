import React from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero', styles.heroBanner)}>
      <div className="container">
        <h1 className="hero__title">Full-featured Claude bridge</h1>
        <p className="hero__subtitle">Simple and easy to use · Chat platform integration · Real-time monitoring</p>
        <div className={styles.buttons}>
          <Link
            className="button button--primary button--lg"
            to="/docs/intro">
            GET STARTED
          </Link>
          <Link
            className="button button--secondary button--lg"
            to="https://github.com/imneov/claudewarp">
            GitHub
          </Link>
        </div>
        <div className="githubButtons">
          <iframe
            src="https://ghbtns.com/github-btn.html?user=imneov&repo=claudewarp&type=star&count=true&size=large"
            frameBorder="0"
            scrolling="0"
            width="170"
            height="30"
            title="GitHub Stars"
          />
          <iframe
            src="https://ghbtns.com/github-btn.html?user=imneov&repo=claudewarp&type=watch&count=true&size=large&v=2"
            frameBorder="0"
            scrolling="0"
            width="170"
            height="30"
            title="GitHub Watchers"
          />
        </div>
      </div>
    </header>
  );
}

export default function Home(): React.ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title} - 智能远程驾驶舱`}
      description="ClaudeWarp 是一个连接 Claude 与聊天平台的智能桥梁，提供实时协作监控和多平台集成能力">
      <HomepageHeader />
      <main>
        <HomepageFeatures />
      </main>
    </Layout>
  );
}
