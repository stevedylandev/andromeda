import clsx from "clsx";
import type { ReactNode } from "react";

export type ButtonProps = {
	children: ReactNode;
	className?: string;
	href?: string;
	variant?: "accent";
	type?: "button" | "submit" | "reset" | "link";
	onClick?: (event: React.MouseEvent<HTMLButtonElement>) => void; // Add this
	onMouseOver?: (event: React.MouseEvent<HTMLButtonElement>) => void; // Add this too for hover
	onMouseOut?: (event: React.MouseEvent<HTMLButtonElement>) => void; // And this
};

export function Button({
	children,
	className,
	href,
	variant,
	type = "button",
}: ButtonProps) {
	const baseClassName = clsx(
		className,
		"inline-flex items-center justify-center px-4 py-2 text-sm font-medium rounded-md border border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-offset-2 hover:opacity-90",
		variant === "accent"
			? "hover:opacity-90 focus:ring-orange-400 text-zinc-900 bg-[#ffb757]"
			: "bg-zinc-900! text-zinc-200! hover:bg-zinc-800! focus:ring-gray-500!",
	);

	if (type === "link") {
		return (
			<a href={href} className={baseClassName}>
				{children}
			</a>
		);
	}

	return (
		<button
			type={type}
			className={baseClassName}
			{...(type !== "submit" && href ? { href } : {})}
		>
			{children}
		</button>
	);
}
