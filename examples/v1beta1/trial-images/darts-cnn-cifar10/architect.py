import torch
import copy


class Architect():
    """" Architect controls architecture of cell by computing gradients of alphas
    """

    def __init__(self, model, w_momentum, w_weight_decay):
        self.model = model
        self.v_model = copy.deepcopy(model)
        self.w_momentum = w_momentum
        self.w_weight_decay = w_weight_decay

    def virtual_step(self, train_x, train_y, xi, w_optim):
        """
        Compute unrolled weight w' (virtual step)
        Step process:
        1) forward
        2) calculate loss
        3) compute gradient (by backprop)
        4) update gradient

        Args:
            xi: learning rate for virtual gradient step (same as weights lr)
            w_optim: weights optimizer
        """

        # Forward and calculate loss
        # Loss for train with w. L_train(w)
        loss = self.model.loss(train_x, train_y)
        # Compute gradient
        gradients = torch.autograd.grad(loss, self.model.getWeights())

        # Do virtual step (Update gradient)
        # Below opeartions do not need gradient tracking
        with torch.no_grad():
            # dict key is not the value, but the pointer. So original network weight have to
            # be iterated also.
            for w, vw, g in zip(self.model.getWeights(), self.v_model.getWeights(), gradients):
                m = w_optim.state[w].get("momentum_buffer", 0.) * self.w_momentum
                vw.copy_(w - xi * (m + g + self.w_weight_decay * w))

            # Sync alphas
            for a, va in zip(self.model.getAlphas(), self.v_model.getAlphas()):
                va.copy_(a)

    def unrolled_backward(self, train_x, train_y, valid_x, valid_y, xi, w_optim):
        """ Compute unrolled loss and backward its gradients
        Args:
            xi: learning rate for virtual gradient step (same as model lr)
            w_optim: weights optimizer - for virtual step
        """
        # Do virtual step (calc w')
        self.virtual_step(train_x, train_y, xi, w_optim)

        # Calculate unrolled loss
        # Loss for validation with w'. L_valid(w')
        loss = self.v_model.loss(valid_x, valid_y)

        # Calculate gradient
        v_alphas = tuple(self.v_model.getAlphas())
        v_weights = tuple(self.v_model.getWeights())
        v_grads = torch.autograd.grad(loss, v_alphas + v_weights)

        dalpha = v_grads[:len(v_alphas)]
        dws = v_grads[len(v_alphas):]

        hessian = self.compute_hessian(dws, train_x, train_y)

        # Update final gradient = dalpha - xi * hessian
        with torch.no_grad():
            for alpha, da, h in zip(self.model.getAlphas(), dalpha, hessian):
                alpha.grad = da - xi * h

    def compute_hessian(self, dws, train_x, train_y):
        """
        dw = dw' { L_valid(w', alpha) }
        w+ = w + eps * dw
        w- = w - eps * dw
        hessian = (dalpha{ L_train(w+, alpha) } - dalpha{ L_train(w-, alpha) }) / (2*eps)
        eps = 0.01 / ||dw||
        """

        norm = torch.cat([dw.view(-1) for dw in dws]).norm()
        eps = 0.01 / norm

        # w+ = w + eps * dw
        with torch.no_grad():
            for p, dw in zip(self.model.getWeights(), dws):
                p += eps * dw

        loss = self.model.loss(train_x, train_y)
        # dalpha { L_train(w+, alpha) }
        dalpha_positive = torch.autograd.grad(loss, self.model.getAlphas())

        # w- = w - eps * dw
        with torch.no_grad():
            for p, dw in zip(self.model.getWeights(), dws):
                # TODO (andreyvelich): Do we need this * 2.0 ?
                p -= 2. * eps * dw

        loss = self.model.loss(train_x, train_y)
        # dalpha { L_train(w-, alpha) }
        dalpha_negative = torch.autograd.grad(loss, self.model.getAlphas())

        # recover w
        with torch.no_grad():
            for p, dw in zip(self.model.getWeights(), dws):
                p += eps * dw

        hessian = [(p-n) / (2. * eps) for p, n in zip(dalpha_positive, dalpha_negative)]
        return hessian
