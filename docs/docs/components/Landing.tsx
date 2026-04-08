import { Button } from "./Button";

export function Landing() {
	return (
		<main
			className="flex w-full sm:h-[85vh] min-h-[85vh] mt-6 mx-auto flex-col items-center justify-center gap-12 p-4 relative"
			style={{
				backgroundImage: "url('/bg.png')",
				backgroundPosition: "center",
				backgroundRepeat: "no-repeat",
				backgroundSize: "cover",
				zIndex: 0,
			}}
		>
			<div
				className="absolute inset-0 pointer-events-none"
				style={{
					backgroundColor: "rgba(18, 17, 19, 0.7)",
					zIndex: -1,
				}}
			/>

			<div className="flex flex-col items-center gap-12">
				<h1 className="text-center sm:text-6xl text-4xl font-black">
          Andromeda
				</h1>
				<h3 className="text-center sm:text-2xl text-lg font-semibold">
          Minimal, self-hosted personal software in Rust
				</h3>
			</div>
			<div className="flex items-center gap-4">
				<Button type="link" variant="accent" href="/quickstart">
					Get Started
				</Button>
				<Button type="link" href="https://github.com/stevedylandev/andromeda">
					GitHub
				</Button>
			</div>
		</main>
	);
}
