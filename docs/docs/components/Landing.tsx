import { Button } from "./Button";

export function Landing() {
	return (
		<main
			style={{
				display: "flex",
				width: "100%",
				minHeight: "85vh",
				marginTop: "2rem",
				flexDirection: "column",
				alignItems: "center",
				justifyContent: "center",
				gap: "3rem",
				padding: "1rem",
				background: "#121113",
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
		</main>
	);
}
