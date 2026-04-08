import { Button } from "./Button";

export function Landing() {
	return (
		<main
			className="landing"
			style={{
				position: "relative",
				width: "100%",
				background: "#121113 url('/bg.png') center center / cover no-repeat",
			}}
		>
			<div
				style={{
					display: "flex",
					width: "100%",
					minHeight: "100vh",
					flexDirection: "column",
					alignItems: "center",
					justifyContent: "center",
					gap: "3rem",
					padding: "1rem",
					background: "rgba(0, 0, 0, 0.65)",
				}}
			>
				<div
					style={{
						display: "flex",
						flexDirection: "column",
						alignItems: "center",
						gap: "1.5rem",
					}}
				>
					<h1
						style={{
							textAlign: "center",
							fontSize: "48px",
							fontWeight: 700,
							fontFamily: '"Commit Mono", monospace, sans-serif',
							textTransform: "uppercase",
							color: "#ffffff",
						}}
					>
						Andromeda
					</h1>
					<h3
						style={{
							textAlign: "center",
							fontSize: "16px",
							fontWeight: 400,
							fontFamily: '"Commit Mono", monospace, sans-serif',
							color: "#ffffff",
							opacity: 0.7,
						}}
					>
						Minimal, self-hosted personal software in Rust
					</h3>
				</div>
				<div style={{ display: "flex", alignItems: "center", gap: "1rem" }}>
					<Button type="link" href="/quickstart">
						Get Started
					</Button>
					<Button type="link" href="https://github.com/stevedylandev/andromeda">
						GitHub
					</Button>
				</div>
			</div>
		</main>
	);
}
