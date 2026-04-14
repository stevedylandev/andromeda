import { Button } from "./Button";

export function Landing() {
	return (
		<main
			className="landing relative w-full bg-[#121113] bg-[url('/bg.webp')] bg-cover bg-center bg-no-repeat"
		>
			<div className="flex w-full min-h-screen flex-col items-center justify-center gap-12 p-4">
				<div className="flex flex-col items-center gap-6">
					<h1 className="text-center sm:text-[92px] text-[64px] font-normal font-['Commit_Mono',monospace] text-white">
						Andromeda
					</h1>
					<h3 className="text-center text-lg font-normal font-['Commit_Mono',monospace] text-white">
						Minimal, self-hosted, personal software in Rust
					</h3>
				</div>
				<div className="flex items-center gap-4">
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
