import type { ReactNode } from "react";

export type ButtonProps = {
	children: ReactNode;
	className?: string;
	href?: string;
	variant?: "accent";
	type?: "button" | "submit" | "reset" | "link";
	onClick?: (event: React.MouseEvent<HTMLButtonElement>) => void;
};

const baseStyle: React.CSSProperties = {
	display: "inline-flex",
	alignItems: "center",
	justifyContent: "center",
	padding: "0.4rem 0.75rem",
	fontSize: "14px",
	fontFamily: '"Commit Mono", monospace, sans-serif',
	background: "#121113",
	color: "#ffffff",
	border: "1px solid white",
	borderRadius: 0,
	cursor: "pointer",
	textDecoration: "none",
	width: "fit-content",
};

export function Button({
	children,
	href,
	type = "button",
}: ButtonProps) {
	if (type === "link") {
		return (
			<a href={href} style={baseStyle}>
				{children}
			</a>
		);
	}

	return (
		<button type={type} style={baseStyle}>
			{children}
		</button>
	);
}
