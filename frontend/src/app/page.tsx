import Link from 'next/link';

export default function Home() {
  return (
    <main className="container">
      <div className="glass-panel">
        <h1>stock-pulse</h1>
        <p>A arquitetura foi estabelecida. A infraestrutura 100% Dockerizada está no ar.</p>
        <Link href="/dashboard/portfolio" passHref>
          <button className="primary-button">Entrar no Dashboard</button>
        </Link>
      </div>
    </main>
  );
}
